package runtime

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/setup"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
)

var (
	ErrClusterNotFound     = errors.New("DST cluster not found")
	ErrUnsupportedCluster  = errors.New("only Steam SERVER clusters can be launched by the AGMP dedicated runtime")
	ErrMasterShardNotFound = errors.New("Master shard not found")
)

type StartMasterRequest struct {
	ClusterPath  string `json:"clusterPath"`
	UGCDirectory string `json:"ugcDirectory"`
}

type ProcessRequest struct {
	ClusterPath string `json:"clusterPath"`
	ShardName   string `json:"shardName"`
}

type LogRequest struct {
	ClusterPath string `json:"clusterPath"`
	ShardName   string `json:"shardName"`
	After       uint64 `json:"after"`
	Limit       int    `json:"limit"`
}

type CommandRequest struct {
	ClusterPath string `json:"clusterPath"`
	ShardName   string `json:"shardName"`
	Command     string `json:"command"`
}

type Service struct {
	workspace *dstworkspace.Service
	setup     *dstsetup.Service
	manager   *Manager
}

func New(workspace *dstworkspace.Service, setup *dstsetup.Service, manager *Manager) *Service {
	return &Service{workspace: workspace, setup: setup, manager: manager}
}

func (s *Service) StartMaster(ctx context.Context, request StartMasterRequest) (ProcessSnapshot, error) {
	return s.StartShard(ctx, StartShardRequest{
		ClusterPath:  request.ClusterPath,
		ShardName:    "Master",
		UGCDirectory: request.UGCDirectory,
	})
}

func (s *Service) Lookup(request ProcessRequest) LookupResult {
	shard := strings.TrimSpace(request.ShardName)
	if shard == "" {
		shard = "Master"
	}
	return s.manager.Lookup(request.ClusterPath, shard)
}

func (s *Service) ReadLogs(request LogRequest) (LogBatch, error) {
	shard := strings.TrimSpace(request.ShardName)
	if shard == "" {
		shard = "Master"
	}
	process := s.manager.Get(request.ClusterPath, shard)
	if process == nil {
		return LogBatch{Lines: []LogLine{}, NextCursor: request.After}, nil
	}
	return process.ReadLogs(request.After, request.Limit), nil
}

func (s *Service) SendCommand(request CommandRequest) (ProcessSnapshot, error) {
	shard := strings.TrimSpace(request.ShardName)
	if shard == "" {
		shard = "Master"
	}
	process := s.manager.Get(request.ClusterPath, shard)
	if process == nil {
		return ProcessSnapshot{}, ErrProcessNotRunning
	}
	if strings.TrimSpace(request.Command) == "" {
		return process.Snapshot(), errors.New("控制台命令不能为空")
	}
	if err := process.SendCommand(request.Command); err != nil {
		return process.Snapshot(), err
	}
	return process.Snapshot(), nil
}

func (s *Service) Stop(request ProcessRequest) (ProcessSnapshot, error) {
	shard := strings.TrimSpace(request.ShardName)
	if shard == "" {
		shard = "Master"
	}
	if strings.EqualFold(shard, "Master") {
		caves := s.manager.Lookup(request.ClusterPath, "Caves")
		if isLookupActive(caves) {
			return ProcessSnapshot{}, errors.New("Caves 仍在运行；请先停止 Caves，或使用“全部优雅停止”按 Caves → Master 顺序关闭")
		}
	}
	process := s.manager.Get(request.ClusterPath, shard)
	if process == nil {
		return ProcessSnapshot{}, ErrProcessNotRunning
	}
	if !s.manager.Stop(request.ClusterPath, shard, DefaultStopOptions()) {
		return process.Snapshot(), nil
	}
	return process.Snapshot(), nil
}

func (s *Service) StopAllBlocking() {
	options := DefaultStopOptions()
	// Preserve shard dependency during application shutdown as well: secondaries
	// stop first, then Master. This keeps the master endpoint alive while Caves
	// performs its final save/shutdown sequence.
	for _, snapshot := range s.manager.Snapshots() {
		if strings.EqualFold(snapshot.ShardName, "Caves") && isLookupActive(LookupResult{Found: true, Process: snapshot}) {
			s.manager.StopBlocking(snapshot.ClusterPath, snapshot.ShardName, options)
		}
	}
	for _, snapshot := range s.manager.Snapshots() {
		if strings.EqualFold(snapshot.ShardName, "Master") && isLookupActive(LookupResult{Found: true, Process: snapshot}) {
			s.manager.StopBlocking(snapshot.ClusterPath, snapshot.ShardName, options)
		}
	}
}

func (s *Service) AnyRunning() bool {
	return s.manager.AnyRunning()
}

func findCluster(clusters []dstdomain.Cluster, wanted string) (dstdomain.Cluster, bool) {
	wanted = filepath.Clean(wanted)
	for _, cluster := range clusters {
		if samePath(cluster.Path, wanted) {
			return cluster, true
		}
	}
	return dstdomain.Cluster{}, false
}

func findMasterShard(cluster dstdomain.Cluster) (dstdomain.Shard, bool) {
	for _, shard := range cluster.Shards {
		if strings.EqualFold(shard.Name, "Master") {
			return shard, true
		}
	}
	return dstdomain.Shard{}, false
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	if strings.EqualFold(filepath.VolumeName(left), filepath.VolumeName(right)) {
		// EqualFold is required on Windows; it is harmless for the ASCII-heavy
		// Klei paths normally seen on other platforms.
		return strings.EqualFold(left, right)
	}
	return left == right
}

func (s *Service) DescribeStartability(ctx context.Context, clusterPath string) error {
	snapshot, err := s.workspace.Snapshot(ctx)
	if err != nil {
		return err
	}
	cluster, ok := findCluster(snapshot.Environment.Clusters, clusterPath)
	if !ok {
		return ErrClusterNotFound
	}
	if cluster.Distribution != dstdomain.DistributionSteam || cluster.Source != dstdomain.SaveSourceServer {
		return ErrUnsupportedCluster
	}
	if _, ok := findMasterShard(cluster); !ok {
		return ErrMasterShardNotFound
	}
	if !snapshot.Dedicated.Installation.Valid {
		return fmt.Errorf("未检测到有效的 DST Dedicated Server 安装")
	}
	if !snapshot.Dedicated.ConfDir.Valid {
		return fmt.Errorf("Klei -conf_dir 不可用: %s", snapshot.Dedicated.ConfDir.Error)
	}
	return nil
}
