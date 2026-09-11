package app

import (
	"context"
	"fmt"
	"io"

	steammaintenance "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/steam/maintenance"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/logcenter"
	dstprefs "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	dstruntimecore "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
	dstworkspace "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst"
	clusterops "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/cluster"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/kleiarchive"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/preflight"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
	platformhttp "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/http"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
	gameworkspace "github.com/yubboo/AI-Game-Manager-Panel/internal/server/workspace"
)

func (a *Application) SteamEnvironment() (steam.Environment, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	env, err := a.steam.Detect(ctx)
	if err != nil {
		a.logger.Warn("Steam environment detection failed", "error", err.Error())
		return steam.Environment{}, err
	}
	if env.Detected {
		a.logger.Info("Steam environment detected", "path", env.InstallPath, "libraries", len(env.Libraries))
	} else {
		a.logger.Info("Steam environment not detected")
	}
	return env, nil
}

// SteamApps scans every appmanifest_*.acf across detected Steam libraries.
// One malformed manifest is reported as a warning and does not abort the inventory.
func (a *Application) SteamApps() (steam.AppInventory, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	inventory, err := a.steam.Apps(ctx)
	if err != nil {
		a.logger.Warn("Steam app inventory scan failed", "error", err.Error())
		return steam.AppInventory{}, err
	}
	a.logger.Info("Steam app inventory scanned", "apps", len(inventory.Apps), "warnings", len(inventory.Warnings))
	return inventory, nil
}

// SteamSnapshot refreshes Steam environment and AppManifest inventory in one
// backend operation. Keeping the snapshot atomic prevents UI flicker caused by
// independent environment/inventory refreshes completing at different times.
func (a *Application) SteamSnapshot() (steam.Snapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot, err := a.steam.Snapshot(ctx)
	if err != nil {
		a.logger.Warn("Steam snapshot refresh failed", "error", err.Error())
		return steam.Snapshot{}, err
	}
	a.logger.Info("Steam snapshot refreshed",
		"detected", snapshot.Environment.Detected,
		"libraries", len(snapshot.Environment.Libraries),
		"apps", len(snapshot.Inventory.Apps),
		"duration_ms", snapshot.DurationMs,
	)
	return snapshot, nil
}

// GameWorkspaceSnapshot refreshes exactly one selected game workspace.
// Steam detection is targeted to the AppIDs required by that game only.
func (a *Application) GameWorkspaceSnapshot(id string) (gameworkspace.Snapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	snapshot, err := a.gameWorkspace.Snapshot(ctx, game.ID(id))
	if err != nil {
		a.logger.Warn("game workspace refresh failed", "game", id, "error", err.Error())
		return gameworkspace.Snapshot{}, err
	}
	a.logger.Info("game workspace refreshed", "game", id, "duration_ms", snapshot.DurationMs)
	return snapshot, nil
}

// DSTEnvironment ports the Python baseline's environment discovery into the
// shared Go core. Both Wails and HTTP adapters expose this exact method.
func (a *Application) DSTEnvironment() dstdomain.Environment {
	env := dstdomain.DiscoverEnvironment()
	a.logger.Info("DST environment discovered",
		"steam_clusters", countDSTClusters(env, dstdomain.DistributionSteam),
		"wegame_clusters", countDSTClusters(env, dstdomain.DistributionWeGame),
	)
	return env
}

// DSTWorkspaceSnapshot returns the complete DST game workspace in one backend
// refresh. It performs one targeted Steam scan and reuses that environment for
// strict Dedicated Server validation.
func (a *Application) DSTWorkspaceSnapshot() (dstworkspace.Snapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstWorkspace.Snapshot(ctx)
	if err != nil {
		a.logger.Warn("DST workspace refresh failed", "error", err.Error())
		return dstworkspace.Snapshot{}, err
	}
	a.logger.Info("DST workspace refreshed",
		"game_installed", value.Workspace.Game.GameApp != nil && value.Workspace.Game.GameApp.Installed,
		"server_valid", value.Dedicated.Installation.Valid,
		"server_arch", value.Dedicated.Installation.Architecture,
	)
	return value, nil
}

// DSTDedicatedServer returns the validated Dedicated Server installation and
// -conf_dir state used by both Desktop and Web. This is a behavior-port of the
// pure discovery/argument portion of DSTCamp dedicated_server.py.
func (a *Application) DSTDedicatedServer() (dedicated.Snapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstDedicated.Snapshot(ctx)
	if err != nil {
		a.logger.Warn("DST dedicated server discovery failed", "error", err.Error())
		return dedicated.Snapshot{}, err
	}
	a.logger.Info("DST dedicated server discovered",
		"valid", value.Installation.Valid,
		"path", value.Installation.RootDir,
		"architecture", value.Installation.Architecture,
		"source", value.Installation.Source,
	)
	return value, nil
}

func (a *Application) DSTDedicatedPreferences() dstprefs.Value {
	return a.dstDedicated.Preferences()
}

func (a *Application) SaveDSTDedicatedPreferences(value dstprefs.Value) error {
	if err := a.dstDedicated.SavePreferences(value); err != nil {
		a.logger.Error("save DST dedicated preferences failed", "error", err.Error())
		return err
	}
	a.logger.Info("DST dedicated preferences saved",
		"manual_path", value.DedicatedServerPath,
		"extra_args_set", value.DedicatedServerExtraArgs != "",
	)
	return nil
}

func (a *Application) BuildDSTLaunchSpec(request dedicated.LaunchRequest) (dedicated.LaunchSpec, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstDedicated.BuildLaunchSpec(ctx, request)
	if err != nil {
		a.logger.Warn("build DST launch spec failed", "cluster", request.ClusterName, "shard", request.ShardName, "error", err.Error())
		return dedicated.LaunchSpec{}, err
	}
	return value, nil
}

func (a *Application) OpenDSTTokenPage() error {
	if err := platformhttp.OpenExternalURL(token.OfficialServerPage); err != nil {
		a.logger.Warn("open Klei token page failed", "error", err.Error())
		return err
	}
	a.logger.Info("opened Klei official token page")
	return nil
}

func (a *Application) DSTTokenStatus(clusterPath string) (token.Status, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.dstSetup.TokenStatus(ctx, clusterPath)
}

func (a *Application) SaveDSTToken(request token.SaveRequest) (token.Status, error) {
	if a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return token.Status{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再替换服务器令牌")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstSetup.SaveToken(ctx, request)
	if err != nil {
		a.logger.Warn("save DST token failed", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	// Never log request.Value or any token-derived content.
	a.logger.Info("DST token saved", "cluster_path", request.ClusterPath, "configured", value.Configured)
	a.recordOperation("info", "dst", "save_token", request.ClusterPath, "success", "已配置 Klei 服务器令牌（正文未记录）", "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) ImportDSTToken(request token.ImportRequest) (token.Status, error) {
	if a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return token.Status{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再替换服务器令牌")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstSetup.ImportToken(ctx, request)
	if err != nil {
		a.logger.Warn("import DST token failed", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST token imported", "cluster_path", request.ClusterPath, "configured", value.Configured)
	a.recordOperation("info", "dst", "import_token", request.ClusterPath, "success", "已导入 Klei 服务器令牌（正文未记录）", "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) InspectDSTKleiPackage(request kleiarchive.InspectRequest) (kleiarchive.Preview, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstSetup.InspectKleiPackage(ctx, request)
	if err != nil {
		a.logger.Warn("inspect Klei config package failed", "archive_name", request.ArchiveName, "error", err.Error())
	}
	return value, err
}

func (a *Application) ImportDSTKleiPackage(request kleiarchive.ImportRequest) (kleiarchive.ImportResult, error) {
	if a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return kleiarchive.ImportResult{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再导入 Klei 配置包")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstSetup.ImportKleiPackage(ctx, request)
	if err != nil {
		a.logger.Warn("import Klei config package failed", "cluster_path", request.ClusterPath, "archive_name", request.ArchiveName, "mode", request.Mode, "error", err.Error())
		return value, err
	}
	// Never log archiveBase64 or any token-derived content.
	a.logger.Info("Klei config package imported", "cluster_path", request.ClusterPath, "archive_name", request.ArchiveName, "mode", value.Mode, "config_files", value.ConfigFilesCopied, "token_configured", value.TokenConfigured)
	a.recordOperation("info", "dst", "import_klei_package", request.ClusterPath, "success", fmt.Sprintf("导入 Klei 配置包，模式=%s，配置文件=%d", value.Mode, value.ConfigFilesCopied), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) DSTPreflight(clusterPath string) (preflight.Result, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	return a.dstSetup.Preflight(ctx, clusterPath)
}

func (a *Application) ImportDSTCluster(request clusterops.ImportRequest) (clusterops.ImportResult, error) {
	if a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return clusterops.ImportResult{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再导入 Cluster")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstSetup.ImportCluster(ctx, request)
	if err != nil {
		a.logger.Warn("import DST cluster failed", "source_path", request.SourcePath, "target_name", request.TargetName, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST cluster imported", "source_path", request.SourcePath, "target_path", value.Path, "copied_files", value.CopiedFiles, "token_skipped", value.TokenSkipped)
	a.recordOperation("info", "dst", "import_cluster", value.Path, "success", fmt.Sprintf("导入世界，复制文件=%d", value.CopiedFiles), "steam.dst", value.Path)
	return value, nil
}

func (a *Application) StartDSTMaster(request dstruntimecore.StartMasterRequest) (dstruntimecore.ProcessSnapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstRuntime.StartMaster(ctx, request)
	if err != nil {
		a.logger.Warn("start DST Master failed", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST Master started", "cluster", value.ClusterName, "pid", value.PID)
	a.recordOperation("info", "dst", "start_shard", value.ClusterName+"/Master", "success", fmt.Sprintf("PID=%d", value.PID), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) StartDSTCluster(request dstruntimecore.StartClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	return a.startDSTCluster(a.Context(), request)
}

// startDSTCluster is the single DST cluster-start domain action shared by the
// human UI and XiaoYu Tool Host. The caller supplies its cancellation context,
// but logging, audit and runtime behavior stay identical for both brains.
func (a *Application) startDSTCluster(ctx context.Context, request dstruntimecore.StartClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstRuntime.StartCluster(ctx, request)
	if err != nil {
		a.logger.Warn("start DST cluster failed", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST cluster started", "cluster", value.ClusterName, "overall", value.Overall, "has_caves", value.HasCaves)
	a.recordOperation("info", "dst", "start_cluster", value.ClusterName, "success", fmt.Sprintf("overall=%s hasCaves=%t", value.Overall, value.HasCaves), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) StartDSTShard(request dstruntimecore.StartShardRequest) (dstruntimecore.ProcessSnapshot, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.dstRuntime.StartShard(ctx, request)
	if err != nil {
		a.logger.Warn("start DST shard failed", "cluster_path", request.ClusterPath, "shard", request.ShardName, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST shard started", "cluster", value.ClusterName, "shard", value.ShardName, "pid", value.PID)
	a.recordOperation("info", "dst", "start_shard", value.ClusterName+"/"+value.ShardName, "success", fmt.Sprintf("PID=%d", value.PID), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) DSTClusterStatus(request dstruntimecore.ClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	return a.dstClusterStatus(a.Context(), request)
}

func (a *Application) dstClusterStatus(ctx context.Context, request dstruntimecore.ClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return a.dstRuntime.ClusterStatus(ctx, request)
}

func (a *Application) DSTPortStatus(request dstruntimecore.ClusterRequest) (dstruntimecore.PortReport, error) {
	return a.dstRuntime.PortStatus(request)
}

func (a *Application) DSTPortConfiguration(request dstruntimecore.ClusterRequest) (dedicated.PortConfiguration, error) {
	return a.dstRuntime.PortConfiguration(request)
}

func (a *Application) ConfigureDSTPorts(request dstruntimecore.PortConfigureRequest) (dstruntimecore.PortConfigureResult, error) {
	value, err := a.dstRuntime.ConfigurePorts(request)
	if err != nil {
		a.logger.Warn("configure DST ports failed", "cluster_path", request.ClusterPath, "recommended", request.UseRecommended, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST ports configured", "cluster_path", request.ClusterPath, "recommended", request.UseRecommended, "backup", value.BackupPath)
	a.recordOperation("info", "dst", "configure_ports", request.ClusterPath, "success", fmt.Sprintf("recommended=%t", request.UseRecommended), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) CleanupDSTPorts(request dstruntimecore.PortCleanupRequest) (dstruntimecore.PortCleanupResult, error) {
	value, err := a.dstRuntime.CleanupPorts(request)
	if err != nil {
		a.logger.Warn("cleanup DST ports incomplete", "cluster_path", request.ClusterPath, "force", request.Force, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST ports cleanup completed", "cluster_path", request.ClusterPath, "terminated_pids", fmt.Sprint(value.TerminatedPIDs), "skipped_pids", fmt.Sprint(value.SkippedPIDs))
	a.recordOperation("warning", "dst", "cleanup_ports", request.ClusterPath, "success", fmt.Sprintf("terminated=%d skipped=%d", len(value.TerminatedPIDs), len(value.SkippedPIDs)), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) StopDSTCluster(request dstruntimecore.ClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	value, err := a.dstRuntime.StopCluster(request)
	if err != nil {
		a.logger.Warn("stop DST cluster incomplete", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST cluster stopped", "cluster_path", request.ClusterPath)
	a.recordOperation("info", "dst", "stop_cluster", request.ClusterPath, "success", "停止地面/洞穴", "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) DSTProcessStatus(request dstruntimecore.ProcessRequest) dstruntimecore.LookupResult {
	return a.dstRuntime.Lookup(request)
}

func (a *Application) DSTProcessLogs(request dstruntimecore.LogRequest) (dstruntimecore.LogBatch, error) {
	return a.dstRuntime.ReadLogs(request)
}

func (a *Application) SendDSTCommand(request dstruntimecore.CommandRequest) (dstruntimecore.ProcessSnapshot, error) {
	value, err := a.dstRuntime.SendCommand(request)
	if err != nil {
		a.logger.Warn("send DST command failed", "cluster_path", request.ClusterPath, "shard", request.ShardName, "error", err.Error())
		return value, err
	}
	a.recordOperation("info", "dst", "send_command", request.ClusterPath+"/"+request.ShardName, "success", fmt.Sprintf("command_length=%d", len(request.Command)), "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) StopDSTProcess(request dstruntimecore.ProcessRequest) (dstruntimecore.ProcessSnapshot, error) {
	value, err := a.dstRuntime.Stop(request)
	if err != nil {
		a.logger.Warn("stop DST process failed", "cluster_path", request.ClusterPath, "shard", request.ShardName, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST process stop requested", "cluster_path", request.ClusterPath, "shard", request.ShardName)
	a.recordOperation("info", "dst", "stop_shard", request.ClusterPath+"/"+request.ShardName, "success", "停止单个 Shard", "steam.dst", request.ClusterPath)
	return value, nil
}

func (a *Application) DSTLogSessions(request logcenter.ListRequest) ([]logcenter.Session, error) {
	return a.dstLogs.List(request)
}

func (a *Application) DSTLogRead(request logcenter.ReadRequest) (logcenter.ReadPage, error) {
	return a.dstLogs.Read(request)
}

func (a *Application) DSTLogTail(request logcenter.TailRequest) (logcenter.ReadPage, error) {
	return a.dstLogs.Tail(request)
}

func (a *Application) DSTLogSearch(request logcenter.SearchRequest) (logcenter.SearchResult, error) {
	return a.dstLogs.Search(request)
}

func (a *Application) DSTLogDiagnostics(id string) (logcenter.Diagnostics, error) {
	return a.dstLogs.Diagnostics(id)
}

func (a *Application) ExportDSTLog(id string) (logcenter.ExportResult, error) {
	value, err := a.dstLogs.Export(id)
	if err != nil {
		a.logger.Warn("export DST log failed", "session", id, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST log exported", "session", id, "path", value.Path)
	return value, nil
}

func (a *Application) DSTLogFile(id string) (logcenter.FileRef, error) {
	return a.dstLogs.File(id)
}

func (a *Application) DSTLogBundleName(request logcenter.BundleRequest) (string, error) {
	return a.dstLogs.BundleName(request)
}

func (a *Application) StreamDSTLogBundle(request logcenter.BundleRequest, writer io.Writer) (string, error) {
	return a.dstLogs.StreamBundle(request, writer)
}

func (a *Application) ExportDSTLogBundle(request logcenter.BundleRequest) (logcenter.ExportResult, error) {
	value, err := a.dstLogs.Bundle(request)
	if err != nil {
		a.logger.Warn("export DST log bundle failed", "cluster_path", request.ClusterPath, "error", err.Error())
		return value, err
	}
	a.logger.Info("DST log bundle exported", "cluster_path", request.ClusterPath, "path", value.Path)
	return value, nil
}

func (a *Application) StartSteamInstall(request steammaintenance.InstallRequest) (steammaintenance.TaskSnapshot, error) {
	if dst.RequiresStoppedRuntimeForSteamMaintenance(request.AppID) && a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return steammaintenance.TaskSnapshot{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再安装或修复文件")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.steamMaintenance.StartInstall(ctx, request.AppID)
	if err != nil {
		a.logger.Warn("start Steam install failed", "appid", request.AppID, "error", err.Error())
		return value, err
	}
	a.logger.Info("Steam install requested", "appid", request.AppID, "task", value.ID)
	a.recordOperation("info", "steam", "install", fmt.Sprint(request.AppID), "success", "已创建 Steam 安装任务", "", "")
	return value, nil
}

func (a *Application) StartSteamValidation(request steammaintenance.ValidateRequest) (steammaintenance.TaskSnapshot, error) {
	if dst.RequiresStoppedRuntimeForSteamMaintenance(request.AppID) && a.dstRuntime != nil && a.dstRuntime.AnyRunning() {
		return steammaintenance.TaskSnapshot{}, fmt.Errorf("Dedicated Server 正在运行，请先正常停止 Master/Caves 后再校验文件")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	value, err := a.steamMaintenance.StartValidate(ctx, request.AppID)
	if err != nil {
		a.logger.Warn("start Steam validation failed", "appid", request.AppID, "error", err.Error())
		return value, err
	}
	a.logger.Info("Steam validation requested", "appid", request.AppID, "task", value.ID)
	a.recordOperation("info", "steam", "validate", fmt.Sprint(request.AppID), "success", "已创建 Steam 校验任务", "", "")
	return value, nil
}

func (a *Application) SteamMaintenanceTask(id string) (steammaintenance.TaskSnapshot, error) {
	value, ok := a.steamMaintenance.Get(id)
	if !ok {
		return steammaintenance.TaskSnapshot{}, fmt.Errorf("Steam maintenance task not found: %s", id)
	}
	return value, nil
}

func countDSTClusters(env dstdomain.Environment, distribution dstdomain.Distribution) int {
	count := 0
	for _, cluster := range env.Clusters {
		if cluster.Distribution == distribution {
			count++
		}
	}
	return count
}

// Games exposes the registry to application services. Game modules will register here in later versions.
func (a *Application) Games() *game.Registry {
	return a.games
}

// LicenseStatus 返回本机许可证和机器码状态。
