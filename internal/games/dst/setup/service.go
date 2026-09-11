package dstsetup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
	clusterops "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/cluster"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/kleiarchive"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/preflight"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
)

type Service struct{ workspace *dstworkspace.Service }

func New(workspace *dstworkspace.Service) *Service { return &Service{workspace: workspace} }

func (s *Service) TokenStatus(ctx context.Context, clusterPath string) (token.Status, error) {
	cluster, _, err := s.serverCluster(ctx, clusterPath)
	if err != nil {
		return token.Status{}, err
	}
	return token.Inspect(cluster.Path)
}

func (s *Service) SaveToken(ctx context.Context, request token.SaveRequest) (token.Status, error) {
	cluster, _, err := s.serverCluster(ctx, request.ClusterPath)
	if err != nil {
		return token.Status{}, err
	}
	return token.Save(cluster.Path, request.Value)
}

func (s *Service) ImportToken(ctx context.Context, request token.ImportRequest) (token.Status, error) {
	cluster, _, err := s.serverCluster(ctx, request.ClusterPath)
	if err != nil {
		return token.Status{}, err
	}
	return token.Import(cluster.Path, request.SourcePath)
}

func (s *Service) InspectKleiPackage(_ context.Context, request kleiarchive.InspectRequest) (kleiarchive.Preview, error) {
	data, err := kleiarchive.DecodeRequest(request.ArchiveName, request.ArchiveBase64)
	if err != nil {
		return kleiarchive.Preview{}, err
	}
	return kleiarchive.Inspect(request.ArchiveName, data)
}

func (s *Service) ImportKleiPackage(ctx context.Context, request kleiarchive.ImportRequest) (kleiarchive.ImportResult, error) {
	cluster, _, err := s.serverCluster(ctx, request.ClusterPath)
	if err != nil {
		return kleiarchive.ImportResult{}, err
	}
	request.ClusterPath = cluster.Path
	return kleiarchive.Apply(request)
}

func (s *Service) Preflight(ctx context.Context, clusterPath string) (preflight.Result, error) {
	snapshot, err := s.workspace.Snapshot(ctx)
	if err != nil {
		return preflight.Result{}, err
	}
	return s.PreflightWithSnapshot(snapshot, clusterPath)
}

// PreflightWithSnapshot lets the runtime reuse the exact workspace snapshot it
// will launch from. This avoids a second Steam/Klei scan and prevents the
// preflight result from being based on a different filesystem observation.
func (s *Service) PreflightWithSnapshot(snapshot dstworkspace.Snapshot, clusterPath string) (preflight.Result, error) {
	wanted := strings.TrimSpace(clusterPath)
	cluster := findCluster(snapshot.Environment.Clusters, wanted)
	status := token.Status{State: token.StateMissing, Message: "尚未配置 cluster_token.txt"}
	if cluster != nil {
		var err error
		status, err = token.Inspect(cluster.Path)
		if err != nil {
			return preflight.Result{}, err
		}
	}
	return preflight.Evaluate(preflight.Input{Dedicated: snapshot.Dedicated, Cluster: cluster, Wanted: wanted, Token: status}), nil
}

func (s *Service) ImportCluster(ctx context.Context, request clusterops.ImportRequest) (clusterops.ImportResult, error) {
	snapshot, err := s.workspace.Snapshot(ctx)
	if err != nil {
		return clusterops.ImportResult{}, err
	}
	root := snapshot.Environment.KleiRoot
	if strings.TrimSpace(root) == "" {
		root = snapshot.Dedicated.ConfDir.KleiRoot
	}
	if strings.TrimSpace(root) == "" {
		return clusterops.ImportResult{}, errors.New("未检测到可用的 Klei/DoNotStarveTogether 根目录")
	}
	result, err := clusterops.Import(root, request)
	if err != nil {
		return clusterops.ImportResult{}, err
	}
	// Newly imported LOCAL worlds often omit the Steam internal ports because the
	// normal game client can supply defaults at runtime. A dedicated two-Shard
	// setup cannot safely let Master and Caves inherit the same implicit values.
	// AGMP therefore normalizes only incomplete/invalid imported port plans.
	if configuration, configErr := dedicated.ReadPortConfiguration(result.Path); configErr == nil && !configuration.Valid {
		settings := configuration.Effective
		if validateErr := dedicated.ValidatePortSettings(settings, configuration.HasCaves); validateErr != nil {
			settings = configuration.Recommended
		}
		if _, applyErr := dedicated.ApplyPortSettings(result.Path, settings); applyErr != nil {
			result.Warnings = append(result.Warnings, "网络端口自动配置失败，可在“网络与端口”中一键修复："+applyErr.Error())
		} else {
			result.Warnings = append(result.Warnings, "AGMP 已自动补齐 Dedicated Server 的 Master/Caves 推荐端口；可在“网络与端口”中自定义。")
		}
	}
	return result, nil
}

func (s *Service) serverCluster(ctx context.Context, path string) (dstdomain.Cluster, dstworkspace.Snapshot, error) {
	snapshot, err := s.workspace.Snapshot(ctx)
	if err != nil {
		return dstdomain.Cluster{}, snapshot, err
	}
	cluster := findCluster(snapshot.Environment.Clusters, strings.TrimSpace(path))
	if cluster == nil {
		return dstdomain.Cluster{}, snapshot, errors.New("未找到所选 Cluster，请重新检测环境")
	}
	if cluster.Distribution != dstdomain.DistributionSteam || cluster.Source != dstdomain.SaveSourceServer {
		return dstdomain.Cluster{}, snapshot, fmt.Errorf("令牌只能应用到 Steam SERVER Cluster；当前为 %s/%s", cluster.Distribution, cluster.Source)
	}
	return *cluster, snapshot, nil
}

func findCluster(clusters []dstdomain.Cluster, wanted string) *dstdomain.Cluster {
	wanted = filepath.Clean(wanted)
	for i := range clusters {
		candidate := filepath.Clean(clusters[i].Path)
		if strings.EqualFold(candidate, wanted) {
			return &clusters[i]
		}
	}
	return nil
}
