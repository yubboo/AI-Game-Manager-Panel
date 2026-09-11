package runtime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
	platformnet "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/net"
)

type StartClusterRequest struct {
	ClusterPath  string `json:"clusterPath"`
	UGCDirectory string `json:"ugcDirectory"`
}

type StartShardRequest struct {
	ClusterPath  string `json:"clusterPath"`
	ShardName    string `json:"shardName"`
	UGCDirectory string `json:"ugcDirectory"`
}

type ClusterRequest struct {
	ClusterPath string `json:"clusterPath"`
}

type PortOwner struct {
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	PID            int    `json:"pid"`
	ProcessPath    string `json:"processPath"`
	ProcessName    string `json:"processName"`
	Managed        bool   `json:"managed"`
	ManagedCluster string `json:"managedCluster"`
	ManagedShard   string `json:"managedShard"`
}

type PortCheck struct {
	Port     int                 `json:"port"`
	Uses     []dedicated.PortUse `json:"uses"`
	Occupied bool                `json:"occupied"`
	Owners   []PortOwner         `json:"owners"`
}

type PortReport struct {
	Ready     bool               `json:"ready"`
	Plan      dedicated.PortPlan `json:"plan"`
	Checks    []PortCheck        `json:"checks"`
	Blockers  []string           `json:"blockers"`
	Inspected int64              `json:"inspectedAt"`
}

type ClusterRuntimeSnapshot struct {
	ClusterPath string       `json:"clusterPath"`
	ClusterName string       `json:"clusterName"`
	HasCaves    bool         `json:"hasCaves"`
	Overall     string       `json:"overall"`
	ShardLink   string       `json:"shardLink"`
	Master      LookupResult `json:"master"`
	Caves       LookupResult `json:"caves"`
	Ports       PortReport   `json:"ports"`
}

type PortCleanupRequest struct {
	ClusterPath string `json:"clusterPath"`
	Force       bool   `json:"force"`
}

type PortCleanupResult struct {
	TerminatedPIDs []int      `json:"terminatedPids"`
	SkippedPIDs    []int      `json:"skippedPids"`
	Report         PortReport `json:"report"`
}

type PortConfigureRequest struct {
	ClusterPath    string                 `json:"clusterPath"`
	UseRecommended bool                   `json:"useRecommended"`
	Settings       dedicated.PortSettings `json:"settings"`
}

type PortConfigureResult struct {
	Configuration dedicated.PortConfiguration `json:"configuration"`
	Report        PortReport                  `json:"report"`
	BackupPath    string                      `json:"backupPath"`
}

func (s *Service) PortConfiguration(request ClusterRequest) (dedicated.PortConfiguration, error) {
	clusterPath := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	if clusterPath == "." || clusterPath == "" {
		return dedicated.PortConfiguration{}, errors.New("缺少 Cluster 路径")
	}
	return dedicated.ReadPortConfiguration(clusterPath)
}

func (s *Service) ConfigurePorts(request PortConfigureRequest) (PortConfigureResult, error) {
	clusterPath := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	if clusterPath == "." || clusterPath == "" {
		return PortConfigureResult{}, errors.New("缺少 Cluster 路径")
	}
	current := s.clusterRuntimeSnapshot(clusterPath, nil, PortReport{})
	if isLookupActive(current.Master) || isLookupActive(current.Caves) {
		return PortConfigureResult{}, errors.New("Master/Caves 运行期间不能修改端口，请先正常停止服务器")
	}
	configuration, err := dedicated.ReadPortConfiguration(clusterPath)
	if err != nil {
		return PortConfigureResult{}, err
	}
	settings := request.Settings
	if request.UseRecommended {
		settings = dedicated.RecommendedPortSettings(configuration.HasCaves)
	}
	applied, err := dedicated.ApplyPortSettings(clusterPath, settings)
	if err != nil {
		return PortConfigureResult{}, err
	}
	_, report, err := s.inspectClusterPorts(clusterPath)
	if err != nil {
		return PortConfigureResult{}, err
	}
	return PortConfigureResult{Configuration: applied.Configuration, Report: report, BackupPath: applied.BackupPath}, nil
}

func (s *Service) StartCluster(ctx context.Context, request StartClusterRequest) (ClusterRuntimeSnapshot, error) {
	clusterPath := strings.TrimSpace(request.ClusterPath)
	if clusterPath == "" {
		return ClusterRuntimeSnapshot{}, errors.New("缺少 Cluster 路径")
	}
	current := s.clusterRuntimeSnapshot(clusterPath, nil, PortReport{})
	if isLookupActive(current.Master) || isLookupActive(current.Caves) {
		return current, errors.New("当前 Cluster 已有 Master/Caves 进程运行或正在停止，请先处理现有进程")
	}

	snapshot, cluster, err := s.prepareClusterStart(ctx, clusterPath)
	if err != nil {
		return ClusterRuntimeSnapshot{}, err
	}
	plan, report, err := s.inspectClusterPorts(cluster.Path)
	if err != nil {
		return ClusterRuntimeSnapshot{}, fmt.Errorf("端口启动前检查失败: %w", err)
	}
	if len(plan.Collisions) == 0 && !report.Ready {
		// Safe automatic cleanup is limited to stale processes that this AGMP
		// runtime already owns for the same Cluster. Unknown or external owners
		// are never killed implicitly.
		if cleanup, cleanupErr := s.CleanupPorts(PortCleanupRequest{ClusterPath: cluster.Path, Force: false}); cleanupErr == nil && len(cleanup.TerminatedPIDs) > 0 {
			plan = cleanup.Report.Plan
			report = cleanup.Report
		}
	}
	if len(plan.Collisions) > 0 || !report.Ready {
		return s.clusterRuntimeSnapshot(cluster.Path, &cluster, report), fmt.Errorf("端口启动前检查未通过：%s", strings.Join(report.Blockers, "；"))
	}

	master, ok := findShard(cluster, "Master")
	if !ok {
		return ClusterRuntimeSnapshot{}, ErrMasterShardNotFound
	}
	if _, err := s.startPreparedShard(snapshot, cluster, master, strings.TrimSpace(request.UGCDirectory)); err != nil {
		return s.clusterRuntimeSnapshot(cluster.Path, &cluster, report), err
	}

	if caves, ok := findShard(cluster, "Caves"); ok {
		if _, err := s.startPreparedShard(snapshot, cluster, caves, strings.TrimSpace(request.UGCDirectory)); err != nil {
			// One-click startup should not silently leave a half-started Cluster.
			s.manager.StopBlocking(cluster.Path, master.Name, DefaultStopOptions())
			return s.clusterRuntimeSnapshot(cluster.Path, &cluster, report), fmt.Errorf("Caves 启动失败，已回滚 Master: %w", err)
		}
	}
	return s.ClusterStatus(ctx, ClusterRequest{ClusterPath: cluster.Path})
}

func (s *Service) StartShard(ctx context.Context, request StartShardRequest) (ProcessSnapshot, error) {
	clusterPath := strings.TrimSpace(request.ClusterPath)
	shardName := strings.TrimSpace(request.ShardName)
	if clusterPath == "" || shardName == "" {
		return ProcessSnapshot{}, errors.New("缺少 Cluster 或 Shard")
	}
	snapshot, cluster, err := s.prepareClusterStart(ctx, clusterPath)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	shard, ok := findShard(cluster, shardName)
	if !ok {
		return ProcessSnapshot{}, fmt.Errorf("未找到 Shard: %s", shardName)
	}
	if strings.EqualFold(shard.Name, "Caves") {
		master := s.manager.Lookup(cluster.Path, "Master")
		if !isLookupActive(master) {
			return ProcessSnapshot{}, errors.New("启动 Caves 前必须先启动 Master")
		}
	}
	plan, err := dedicated.ReadPortPlan(cluster.Path)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	uses := portUsesForShard(plan, shard.Name)
	if err := s.requirePortsFree(uses); err != nil {
		return ProcessSnapshot{}, err
	}
	return s.startPreparedShard(snapshot, cluster, shard, strings.TrimSpace(request.UGCDirectory))
}

func (s *Service) prepareClusterStart(ctx context.Context, clusterPath string) (dstworkspace.Snapshot, dstdomain.Cluster, error) {
	snapshot, err := s.workspace.Snapshot(ctx)
	if err != nil {
		return snapshot, dstdomain.Cluster{}, err
	}
	check, err := s.setup.PreflightWithSnapshot(snapshot, clusterPath)
	if err != nil {
		return snapshot, dstdomain.Cluster{}, fmt.Errorf("启动前检查失败: %w", err)
	}
	if !check.Ready {
		problems := make([]string, 0, check.Blockers)
		for _, item := range check.Checks {
			if item.Severity == "blocker" {
				problems = append(problems, item.Label+": "+item.Message)
			}
		}
		return snapshot, dstdomain.Cluster{}, fmt.Errorf("启动前检查未通过：%s", strings.Join(problems, "；"))
	}
	if !snapshot.Dedicated.Installation.Valid {
		return snapshot, dstdomain.Cluster{}, errors.New("未检测到有效的 DST Dedicated Server 安装")
	}
	cluster, ok := findCluster(snapshot.Environment.Clusters, clusterPath)
	if !ok {
		return snapshot, dstdomain.Cluster{}, ErrClusterNotFound
	}
	if cluster.Distribution != dstdomain.DistributionSteam || cluster.Source != dstdomain.SaveSourceServer {
		return snapshot, dstdomain.Cluster{}, ErrUnsupportedCluster
	}
	if err := dedicated.EnsureConsoleEnabled(filepath.Join(cluster.Path, "cluster.ini")); err != nil {
		return snapshot, dstdomain.Cluster{}, fmt.Errorf("启用 DST 控制台失败: %w", err)
	}
	return snapshot, cluster, nil
}

func (s *Service) startPreparedShard(snapshot dstworkspace.Snapshot, cluster dstdomain.Cluster, shard dstdomain.Shard, ugcDirectory string) (ProcessSnapshot, error) {
	spec, err := dedicated.BuildLaunchSpec(snapshot.Dedicated.Installation, snapshot.Dedicated.ConfDir.DocumentsDir, dedicated.LaunchRequest{
		ClusterName:  cluster.Name,
		ShardName:    shard.Name,
		KleiRoot:     snapshot.Environment.KleiRoot,
		UGCDirectory: ugcDirectory,
		ExtraArgs:    snapshot.Dedicated.ExtraArgs,
	})
	if err != nil {
		return ProcessSnapshot{}, err
	}
	process, err := s.manager.Start(StartRequest{ClusterName: cluster.Name, ClusterPath: cluster.Path, ShardName: shard.Name, Spec: spec})
	if err != nil {
		return ProcessSnapshot{}, err
	}
	return process.Snapshot(), nil
}

func (s *Service) ClusterStatus(_ context.Context, request ClusterRequest) (ClusterRuntimeSnapshot, error) {
	clusterPath := filepath.Clean(strings.TrimSpace(request.ClusterPath))
	if clusterPath == "." || clusterPath == "" {
		return ClusterRuntimeSnapshot{}, errors.New("缺少 Cluster 路径")
	}
	cluster := &dstdomain.Cluster{Name: filepath.Base(clusterPath), Path: clusterPath, Shards: []dstdomain.Shard{{Name: "Master", Path: filepath.Join(clusterPath, "Master")}}}
	if info, err := os.Stat(filepath.Join(clusterPath, "Caves", "server.ini")); err == nil && info.Mode().IsRegular() {
		cluster.Shards = append(cluster.Shards, dstdomain.Shard{Name: "Caves", Path: filepath.Join(clusterPath, "Caves")})
	}
	result := s.clusterRuntimeSnapshot(clusterPath, cluster, PortReport{Ready: true, Checks: []PortCheck{}, Blockers: []string{}})
	// Port ownership is useful before startup and while a failed shutdown is
	// converging. During normal running the expected DST processes own these
	// ports, so repeatedly enumerating the Windows port table would add noise and
	// unnecessary work to the 1.2s runtime poll.
	if result.Overall == "stopped" || result.Overall == "crashed" || result.Overall == "stopping" {
		_, report, err := s.inspectClusterPorts(clusterPath)
		if err == nil {
			result.Ports = report
		}
	}
	return result, nil
}

func (s *Service) clusterRuntimeSnapshot(clusterPath string, cluster *dstdomain.Cluster, report PortReport) ClusterRuntimeSnapshot {
	master := s.manager.Lookup(clusterPath, "Master")
	caves := s.manager.Lookup(clusterPath, "Caves")
	name := filepath.Base(filepath.Clean(clusterPath))
	hasCaves := false
	if cluster != nil {
		name = cluster.Name
		_, hasCaves = findShard(*cluster, "Caves")
	}
	return ClusterRuntimeSnapshot{
		ClusterPath: clusterPath,
		ClusterName: name,
		HasCaves:    hasCaves,
		Overall:     overallRuntime(master, caves, hasCaves),
		ShardLink:   shardLinkStatus(master, caves, hasCaves),
		Master:      master,
		Caves:       caves,
		Ports:       report,
	}
}

func shardLinkStatus(master, caves LookupResult, hasCaves bool) string {
	if !hasCaves {
		return "not_applicable"
	}
	if master.Found && caves.Found && master.Process.WorldReady && caves.Process.WorldReady && isLookupActive(master) && isLookupActive(caves) {
		return "ready"
	}
	if isLookupActive(master) && isLookupActive(caves) {
		return "waiting"
	}
	if isLookupActive(master) || isLookupActive(caves) {
		return "disconnected"
	}
	return "stopped"
}

func overallRuntime(master, caves LookupResult, hasCaves bool) string {
	masterActive := isLookupActive(master)
	cavesActive := isLookupActive(caves)
	if !masterActive && !cavesActive {
		if (master.Found && master.Process.Status == dedicated.StatusCrashed) || (caves.Found && caves.Process.Status == dedicated.StatusCrashed) {
			return "crashed"
		}
		return "stopped"
	}
	if hasCaves && masterActive != cavesActive {
		return "partial"
	}
	if (master.Found && master.Process.Status == dedicated.StatusStopping) || (caves.Found && caves.Process.Status == dedicated.StatusStopping) {
		return "stopping"
	}
	if (master.Found && (master.Process.Status == dedicated.StatusStarting || !master.Process.WorldReady)) || (hasCaves && caves.Found && (caves.Process.Status == dedicated.StatusStarting || !caves.Process.WorldReady)) {
		return "starting"
	}
	return "running"
}

func (s *Service) StopCluster(request ClusterRequest) (ClusterRuntimeSnapshot, error) {
	clusterPath := strings.TrimSpace(request.ClusterPath)
	if clusterPath == "" {
		return ClusterRuntimeSnapshot{}, errors.New("缺少 Cluster 路径")
	}
	current := s.clusterRuntimeSnapshot(clusterPath, nil, PortReport{})
	if !isLookupActive(current.Master) && !isLookupActive(current.Caves) {
		return current, ErrProcessNotRunning
	}

	// Stopping can legitimately take tens of seconds while each shard saves.
	// Never block the Wails/HTTP request for that whole interval. Caves is still
	// stopped first, then Master; UI polling observes the transition to stopped.
	go func() {
		s.manager.StopBlocking(clusterPath, "Caves", DefaultStopOptions())
		s.manager.StopBlocking(clusterPath, "Master", DefaultStopOptions())
	}()
	return current, nil
}

func (s *Service) PortStatus(request ClusterRequest) (PortReport, error) {
	_, report, err := s.inspectClusterPorts(strings.TrimSpace(request.ClusterPath))
	return report, err
}

func (s *Service) CleanupPorts(request PortCleanupRequest) (PortCleanupResult, error) {
	clusterPath := strings.TrimSpace(request.ClusterPath)
	if clusterPath == "" {
		return PortCleanupResult{}, errors.New("缺少 Cluster 路径")
	}
	_, report, err := s.inspectClusterPorts(clusterPath)
	if err != nil {
		return PortCleanupResult{}, err
	}
	result := PortCleanupResult{TerminatedPIDs: []int{}, SkippedPIDs: []int{}, Report: report}
	if report.Ready {
		return result, nil
	}
	managed := s.manager.ManagedPIDs()
	seen := map[int]struct{}{}
	for _, check := range report.Checks {
		for _, owner := range check.Owners {
			if owner.PID <= 0 {
				continue
			}
			if _, ok := seen[owner.PID]; ok {
				continue
			}
			seen[owner.PID] = struct{}{}
			if snapshot, ok := managed[owner.PID]; ok {
				if !samePath(snapshot.ClusterPath, clusterPath) {
					// A different AGMP-managed Cluster may intentionally use this
					// process. Never stop another managed server as a side effect.
					result.SkippedPIDs = append(result.SkippedPIDs, owner.PID)
					continue
				}
				if process := s.manager.Get(snapshot.ClusterPath, snapshot.ShardName); process != nil {
					process.StopBlocking(DefaultStopOptions())
					result.TerminatedPIDs = append(result.TerminatedPIDs, owner.PID)
					continue
				}
			}
			if !request.Force {
				result.SkippedPIDs = append(result.SkippedPIDs, owner.PID)
				continue
			}
			// Explicit force-cleanup is intentionally limited to DST Dedicated
			// Server processes. An unrelated program owning the same port must not
			// be killed by a game panel.
			if !isDSTDedicatedProcess(owner.ProcessName, owner.ProcessPath) {
				result.SkippedPIDs = append(result.SkippedPIDs, owner.PID)
				continue
			}
			if err := platformnet.TerminatePID(owner.PID); err != nil {
				result.SkippedPIDs = append(result.SkippedPIDs, owner.PID)
				continue
			}
			result.TerminatedPIDs = append(result.TerminatedPIDs, owner.PID)
		}
	}
	if len(result.TerminatedPIDs) > 0 {
		time.Sleep(500 * time.Millisecond)
	}
	_, result.Report, err = s.inspectClusterPorts(clusterPath)
	if err != nil {
		return result, err
	}
	if !result.Report.Ready && request.Force {
		return result, fmt.Errorf("仍有端口无法安全释放；AGMP 不会结束非 DST 进程：%s", strings.Join(result.Report.Blockers, "；"))
	}
	return result, nil
}

func (s *Service) inspectClusterPorts(clusterPath string) (dedicated.PortPlan, PortReport, error) {
	plan, err := dedicated.ReadPortPlan(clusterPath)
	if err != nil {
		return plan, PortReport{}, err
	}
	statuses, err := platformnet.InspectUDP(plan.Ports())
	if err != nil {
		return plan, PortReport{}, err
	}
	managed := s.manager.ManagedPIDs()
	usesByPort := map[int][]dedicated.PortUse{}
	for _, use := range plan.Uses {
		usesByPort[use.Port] = append(usesByPort[use.Port], use)
	}
	report := PortReport{Ready: len(plan.Collisions) == 0, Plan: plan, Checks: []PortCheck{}, Blockers: []string{}, Inspected: time.Now().Unix()}
	for _, collision := range plan.Collisions {
		report.Blockers = append(report.Blockers, fmt.Sprintf("端口 %d 在配置中被重复使用", collision.Port))
	}
	for _, status := range statuses {
		check := PortCheck{Port: status.Port, Uses: usesByPort[status.Port], Occupied: status.Occupied, Owners: []PortOwner{}}
		udpOccupied := false
		for _, owner := range status.Owners {
			if owner.Protocol != platformnet.ProtocolUDP {
				continue
			}
			udpOccupied = true
			view := PortOwner{Port: owner.Port, Protocol: string(owner.Protocol), PID: owner.PID, ProcessPath: owner.ProcessPath, ProcessName: owner.ProcessName}
			if snapshot, ok := managed[owner.PID]; ok {
				view.Managed = true
				view.ManagedCluster = snapshot.ClusterPath
				view.ManagedShard = snapshot.ShardName
			}
			check.Owners = append(check.Owners, view)
		}
		check.Occupied = udpOccupied
		if udpOccupied {
			report.Ready = false
			ownerText := "未知进程"
			if len(check.Owners) > 0 {
				parts := make([]string, 0, len(check.Owners))
				for _, owner := range check.Owners {
					name := owner.ProcessName
					if name == "" {
						name = "PID"
					}
					parts = append(parts, fmt.Sprintf("%s %d", name, owner.PID))
				}
				ownerText = strings.Join(parts, ", ")
			}
			report.Blockers = append(report.Blockers, fmt.Sprintf("端口 %d 已被占用（%s）", status.Port, ownerText))
		}
		report.Checks = append(report.Checks, check)
	}
	return plan, report, nil
}

func (s *Service) requirePortsFree(uses []dedicated.PortUse) error {
	ports := make([]int, 0, len(uses))
	for _, use := range uses {
		ports = append(ports, use.Port)
	}
	statuses, err := platformnet.InspectUDP(ports)
	if err != nil {
		return fmt.Errorf("端口检查失败: %w", err)
	}
	blocked := []string{}
	for _, status := range statuses {
		udpOwners := make([]platformnet.Owner, 0, len(status.Owners))
		for _, owner := range status.Owners {
			if owner.Protocol == platformnet.ProtocolUDP {
				udpOwners = append(udpOwners, owner)
			}
		}
		if len(udpOwners) == 0 {
			continue
		}
		ownerText := "未知进程"
		if len(udpOwners) > 0 {
			owner := udpOwners[0]
			name := owner.ProcessName
			if name == "" {
				name = "PID"
			}
			ownerText = fmt.Sprintf("%s %d", name, owner.PID)
		}
		blocked = append(blocked, fmt.Sprintf("%d 被 %s 占用", status.Port, ownerText))
	}
	if len(blocked) > 0 {
		return fmt.Errorf("启动所需端口已被占用：%s。若为 AGMP/DST 遗留进程，请先执行端口清理；未知程序不会被自动结束", strings.Join(blocked, "；"))
	}
	return nil
}

func portUsesForShard(plan dedicated.PortPlan, shard string) []dedicated.PortUse {
	result := []dedicated.PortUse{}
	for _, use := range plan.Uses {
		if strings.EqualFold(use.ShardName, shard) {
			result = append(result, use)
		}
	}
	return result
}

func findShard(cluster dstdomain.Cluster, wanted string) (dstdomain.Shard, bool) {
	for _, shard := range cluster.Shards {
		if strings.EqualFold(shard.Name, wanted) {
			return shard, true
		}
	}
	return dstdomain.Shard{}, false
}

func isLookupActive(value LookupResult) bool {
	if !value.Found {
		return false
	}
	return value.Process.Status == dedicated.StatusStarting || value.Process.Status == dedicated.StatusRunning || value.Process.Status == dedicated.StatusStopping
}

func isDSTDedicatedProcess(name, path string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	path = strings.ToLower(strings.TrimSpace(path))
	return strings.Contains(name, "dontstarve_dedicated_server") || strings.Contains(path, "dontstarve_dedicated_server")
}
