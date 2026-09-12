package minecraft

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	environment "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	serverinstance "github.com/yubboo/AI-Game-Manager-Panel/internal/server/instance"
)

type Service struct {
	root          string
	instancesRoot string
	client        *http.Client
	environment   *environment.Service
	store         *serverinstance.Store
	resolver      Resolver
	runtime       *runtimeManager
}

type Options struct {
	Root, InstancesRoot string
	HTTPClient          *http.Client
	Environment         *environment.Service
	Store               *serverinstance.Store
	Resolver            Resolver
}

func New(options Options) *Service {
	c := options.HTTPClient
	if c == nil {
		c = &http.Client{Timeout: 90 * time.Second}
	}
	resolver := options.Resolver
	if resolver.Client == nil {
		resolver.Client = c
	}
	return &Service{root: filepath.Clean(options.Root), instancesRoot: filepath.Clean(options.InstancesRoot), client: c, environment: options.Environment, store: options.Store, resolver: resolver, runtime: newRuntimeManager()}
}

var slugRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func slugName(v string) string {
	v = strings.Trim(strings.ToLower(slugRE.ReplaceAllString(strings.TrimSpace(v), "-")), "-._")
	if v == "" {
		v = "minecraft"
	}
	if len(v) > 48 {
		v = v[:48]
	}
	return v
}

func (s *Service) Plan(ctx context.Context, req PlanRequest) (Plan, error) {
	if req.Software == "" {
		req.Software = SoftwarePaper
	}
	if req.OnlineMode == nil {
		return Plan{}, errors.New("必须明确 Minecraft online-mode；不能把未填写默认为离线模式")
	}
	if !*req.OnlineMode && !req.Whitelist {
		return Plan{}, errors.New("offline 模式必须启用白名单；0.3.0 拒绝无白名单的离线服")
	}
	if req.MemoryMB == 0 {
		req.MemoryMB = 4096
	}
	if req.MemoryMB < 512 || req.MemoryMB > 131072 {
		return Plan{}, errors.New("Minecraft 内存必须在 512MB 到 131072MB 之间")
	}
	if req.Port == 0 {
		req.Port = 25565
	}
	if req.Port < 1 || req.Port > 65535 {
		return Plan{}, errors.New("Minecraft 端口无效")
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "Minecraft Server"
	}
	if req.Origin == "" {
		req.Origin = serverinstance.OriginVisual
	}
	facts, err := s.resolver.Resolve(ctx, req.Version, req.Software)
	if err != nil {
		return Plan{}, err
	}
	target := filepath.Join(s.instancesRoot, "minecraft", slugName(req.Name))
	id := serverinstance.StableID(GameID, target)
	steps := []string{"实时查证 Mojang 版本与 Java 要求", fmt.Sprintf("解析 %s 官方服务端构建", req.Software), fmt.Sprintf("准备 Java %d Runtime", facts.JavaMajor), "下载服务端并做哈希校验", "生成 EULA 与 server.properties", "写入共享 GameInstance"}
	if req.StartAfterDeploy {
		steps = append(steps, "启动 Native server process", "等待 Done ready marker", "执行 Minecraft status ping")
	}
	return Plan{ID: id, GameID: GameID, Name: req.Name, Origin: req.Origin, InstallPath: target, VersionFacts: facts, MemoryMB: req.MemoryMB, Port: req.Port, OnlineMode: *req.OnlineMode, Whitelist: req.Whitelist, EULAAccepted: req.EULAAccepted, AutoInstallJava: req.AutoInstallJava, StartAfterDeploy: req.StartAfterDeploy, Steps: steps}, nil
}

func (s *Service) Deploy(ctx context.Context, req PlanRequest) (DeploymentResult, error) {
	if !req.EULAAccepted {
		return DeploymentResult{}, errors.New("部署 Minecraft 前必须由用户明确接受 Minecraft EULA")
	}
	plan, err := s.Plan(ctx, req)
	if err != nil {
		return DeploymentResult{}, err
	}
	if _, err := os.Stat(plan.InstallPath); err == nil {
		return DeploymentResult{}, fmt.Errorf("实例目录已存在：%s；当前版本拒绝静默覆盖", plan.InstallPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return DeploymentResult{}, err
	}
	if plan.StartAfterDeploy {
		if err := ensurePortAvailable(plan.Port); err != nil {
			return DeploymentResult{}, err
		}
	}
	if s.environment == nil || s.store == nil {
		return DeploymentResult{}, errors.New("Minecraft deploy service 未初始化 Runtime/Instance Store")
	}
	java, err := s.environment.ResolveRuntime(environment.ResolveRuntimeRequest{Kind: environment.RuntimeJava, Major: plan.VersionFacts.JavaMajor})
	if err != nil && plan.AutoInstallJava {
		java, err = s.environment.InstallJava(ctx, environment.JavaInstallRequest{Major: plan.VersionFacts.JavaMajor})
	}
	if err != nil {
		return DeploymentResult{}, fmt.Errorf("需要 Java %d Runtime：%w", plan.VersionFacts.JavaMajor, err)
	}
	stage := plan.InstallPath + ".stage-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return DeploymentResult{}, err
	}
	defer os.RemoveAll(stage)
	artifact := plan.VersionFacts.Artifact
	serverJar := filepath.Join(stage, "server.jar")
	observed, err := s.download(ctx, artifact, serverJar)
	if err != nil {
		return DeploymentResult{}, err
	}
	if artifact.Hash == "" {
		artifact.HashAlgorithm = "sha256"
		artifact.Hash = observed
		artifact.Trust += "; 本地 SHA256=" + observed[:12]
	}
	if err := os.WriteFile(filepath.Join(stage, "eula.txt"), []byte("# Generated by AGMP after explicit user acceptance\neula=true\n"), 0o644); err != nil {
		return DeploymentResult{}, err
	}
	props := fmt.Sprintf("# Generated by AGMP\nserver-port=%d\nonline-mode=%t\nwhite-list=%t\nenable-rcon=false\nenable-query=false\nmotd=Managed by AGMP\n", plan.Port, plan.OnlineMode, plan.Whitelist)
	if err := os.WriteFile(filepath.Join(stage, "server.properties"), []byte(props), 0o644); err != nil {
		return DeploymentResult{}, err
	}
	m := manifest{Version: 1, InstanceID: plan.ID, GameVersion: plan.VersionFacts.Version, Software: plan.VersionFacts.Artifact.Software, ServerVersion: firstNonEmpty(plan.VersionFacts.Artifact.Build, plan.VersionFacts.Artifact.Loader), JavaMajor: plan.VersionFacts.JavaMajor, JavaRuntimeID: java.ID, JavaExecutable: java.Executable, MemoryMB: plan.MemoryMB, Port: plan.Port, Artifact: artifact}
	raw, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(stage, "agmp-minecraft.json"), append(raw, '\n'), 0o600); err != nil {
		return DeploymentResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(plan.InstallPath), 0o755); err != nil {
		return DeploymentResult{}, err
	}
	if err := platformfiles.AtomicReplace(stage, plan.InstallPath); err != nil {
		return DeploymentResult{}, err
	}
	now := time.Now().Unix()
	inst := serverinstance.Instance{ID: plan.ID, Name: plan.Name, GameID: GameID, Origin: plan.Origin, NodeOS: runtime.GOOS, NodeArch: runtime.GOARCH, InstallPath: plan.InstallPath, RuntimeState: "stopped", DesiredState: "stopped", Health: "unknown", GameVersion: plan.VersionFacts.Version, ServerType: string(plan.VersionFacts.Artifact.Software), ServerVersion: m.ServerVersion, Address: fmt.Sprintf("127.0.0.1:%d", plan.Port), Port: plan.Port, Capabilities: []string{"overview", "console", "config", "network"}, Managed: true, CreatedAt: now, UpdatedAt: now}
	inst, err = s.store.Upsert(inst)
	if err != nil {
		// The install directory only becomes authoritative once GameInstance persistence succeeds.
		// Remove the freshly-created directory on store failure so retry does not get stuck behind
		// the no-overwrite guard with an orphaned partial deployment.
		_ = os.RemoveAll(plan.InstallPath)
		return DeploymentResult{}, err
	}
	result := DeploymentResult{Plan: plan, Instance: inst, Runtime: RuntimeSnapshot{InstanceID: inst.ID, State: "stopped"}}
	if plan.StartAfterDeploy {
		snap, err := s.Start(ctx, inst.ID)
		result.Runtime = snap
		if err != nil {
			return result, err
		}
		readyCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		defer cancel()
		if err := s.waitReady(readyCtx, inst.ID); err != nil {
			return result, s.failDeploymentValidation(&result, inst, err)
		}
		probe, err := Probe(readyCtx, "127.0.0.1", plan.Port)
		if err != nil {
			return result, s.failDeploymentValidation(&result, inst, fmt.Errorf("服务器已输出 Done，但 Minecraft Ping 验证失败: %w", err))
		}
		result.Probe = &probe
		inst.RuntimeState = "running"
		inst.DesiredState = "running"
		inst.Health = "healthy"
		inst.Address = fmt.Sprintf("127.0.0.1:%d", plan.Port)
		inst, _ = s.store.Upsert(inst)
		result.Instance = inst
		result.Runtime = s.Status(inst.ID)
	}
	return result, nil
}

func (s *Service) failDeploymentValidation(result *DeploymentResult, inst serverinstance.Instance, cause error) error {
	snap := s.Status(inst.ID)
	if snap.State == "running" || snap.State == "starting" {
		stopped, stopErr := s.Stop(inst.ID)
		result.Runtime = stopped
		if stopErr != nil {
			cause = fmt.Errorf("%w; 自动停止失败: %v", cause, stopErr)
		}
	} else {
		result.Runtime = snap
	}
	latest, err := s.store.Get(inst.ID)
	if err == nil {
		latest.DesiredState = "stopped"
		latest.RuntimeState = result.Runtime.State
		if latest.RuntimeState == "stopping" {
			latest.RuntimeState = "stopped"
		}
		latest.Health = "unhealthy"
		if saved, saveErr := s.store.Upsert(latest); saveErr == nil {
			result.Instance = saved
		}
	}
	return cause
}

func (s *Service) download(ctx context.Context, a Artifact, target string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.URL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "AGMP/0.3.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载 Minecraft 服务端失败: HTTP %d", resp.StatusCode)
	}
	const max = int64(512 << 20)
	if resp.ContentLength > max {
		return "", errors.New("Minecraft 服务端文件超过 512MiB 安全上限")
	}
	f, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var h hash.Hash
	if a.HashAlgorithm == "sha1" {
		h = sha1.New()
	} else {
		h = sha256.New()
	}
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(resp.Body, max+1))
	if err != nil {
		return "", err
	}
	if n > max {
		return "", errors.New("Minecraft 服务端文件超过 512MiB 安全上限")
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if a.Hash != "" && !strings.EqualFold(sum, a.Hash) {
		return "", fmt.Errorf("Minecraft 服务端 %s 校验失败", a.HashAlgorithm)
	}
	return sum, nil
}
func (s *Service) readManifest(path string) (manifest, error) {
	raw, err := os.ReadFile(filepath.Join(path, "agmp-minecraft.json"))
	if err != nil {
		return manifest{}, err
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return manifest{}, err
	}
	return m, nil
}
func (s *Service) Start(ctx context.Context, id string) (RuntimeSnapshot, error) {
	if p := s.runtime.get(id); p != nil {
		snapshot := p.snapshot()
		if snapshot.State == "running" || snapshot.State == "starting" {
			return snapshot, nil
		}
	}
	inst, err := s.store.Get(id)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	if inst.GameID != GameID {
		return RuntimeSnapshot{}, errors.New("实例不是 Minecraft")
	}
	m, err := s.readManifest(inst.InstallPath)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	if err := ensurePortAvailable(m.Port); err != nil {
		return RuntimeSnapshot{}, err
	}
	spec := platformruntime.Spec{Executable: m.JavaExecutable, Arguments: []string{fmt.Sprintf("-Xms%dM", minInt(1024, m.MemoryMB)), fmt.Sprintf("-Xmx%dM", m.MemoryMB), "-jar", "server.jar", "nogui"}, WorkingDirectory: inst.InstallPath}
	p, err := s.runtime.start(id, spec)
	if err != nil {
		return RuntimeSnapshot{}, err
	}
	inst.RuntimeState = "starting"
	inst.DesiredState = "running"
	inst.Health = "starting"
	_, _ = s.store.Upsert(inst)
	return p.snapshot(), nil
}
func (s *Service) Stop(id string) (RuntimeSnapshot, error) {
	p := s.runtime.get(id)
	if p == nil {
		return RuntimeSnapshot{}, errors.New("Minecraft 实例当前没有受 AGMP 管理的运行进程")
	}
	if err := p.stop(); err != nil {
		return p.snapshot(), err
	}
	inst, err := s.store.Get(id)
	if err == nil {
		inst.RuntimeState = "stopped"
		inst.DesiredState = "stopped"
		inst.Health = "stopped"
		_, _ = s.store.Upsert(inst)
	}
	return p.snapshot(), nil
}
func (s *Service) Status(id string) RuntimeSnapshot {
	if p := s.runtime.get(id); p != nil {
		return p.snapshot()
	}
	return RuntimeSnapshot{InstanceID: id, State: "stopped"}
}
func (s *Service) Logs(id string, after uint64, limit int) LogBatch {
	if p := s.runtime.get(id); p != nil {
		return p.logsAfter(after, limit)
	}
	return LogBatch{Lines: []LogLine{}, NextCursor: after}
}
func (s *Service) ProbeInstance(ctx context.Context, id string) (ProbeResult, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return ProbeResult{}, err
	}
	return Probe(ctx, "127.0.0.1", inst.Port)
}
func (s *Service) waitReady(ctx context.Context, id string) error {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		snap := s.Status(id)
		if snap.Ready {
			return nil
		}
		if snap.State == "failed" || snap.State == "stopped" {
			return fmt.Errorf("Minecraft 服务器启动失败: %s", snap.Error)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待 Minecraft Ready 超时: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

func ensurePortAvailable(port int) error {
	if port < 1 || port > 65535 {
		return errors.New("Minecraft 端口无效")
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("Minecraft 端口 %d 已被占用或不可绑定: %w", port, err)
	}
	return listener.Close()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
