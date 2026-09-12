package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
	environmentservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	steammaintenance "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/steam/maintenance"
	updaterservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/updater"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/logcenter"
	dstprefs "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	dstruntimecore "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
	dstsetup "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/setup"
	dstworkspace "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst"
	opsfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/files"
	globallogs "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/logs"
	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
	steamprotocol "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam/protocol"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/telemetry"
	gameworkspace "github.com/yubboo/AI-Game-Manager-Panel/internal/server/workspace"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	licenseservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/system/settings"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

const (
	// Name 与 Slogan 仅作为 configs/app.json 无法读取时的安全回退。
	// 正常运行时，产品名称与标语均从统一配置中心读取，避免散落硬编码。
	Name    = "AI游戏管理器面板"
	Version = "0.2.12"
	Slogan  = "现代化智能 AI 一键游戏服务器部署与管理平台"
)

// Info is the stable application metadata exposed to adapters such as Wails or a future HTTP API.
type Info struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Slogan    string `json:"slogan"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
	DataDir   string `json:"dataDir"`
}

// Application owns AI Game Manager Panel's application-level services.
// UI adapters must call this layer instead of directly touching files, processes or game runtimes.
type Application struct {
	ctx                context.Context
	dataDir            string
	logger             *telemetry.Logger
	settings           *settings.Store
	platformConfig     config.PlatformConfig
	platformConfigErr  error
	games              *game.Registry
	steam              *steam.Service
	gameWorkspace      *gameworkspace.Service
	dstDedicated       *dedicated.Service
	dstWorkspace       *dstworkspace.Service
	dstRuntime         *dstruntimecore.Service
	dstLogs            *logcenter.Service
	dstSetup           *dstsetup.Service
	logHub             *globallogs.Service
	steamMaintenance   *steammaintenance.Service
	auth               *authservice.Service
	environment        *environmentservice.Service
	license            *licenseservice.Service
	updater            *updaterservice.Service
	xiaoyuRuntime      *xiaoyuruntime.Service
	xiaoyuApprovals    *xiaoyucontrol.Store
	xiaoyuTools        *xiaoyucontract.Registry
	xiaoyuHost         *xiaoyuhost.Kernel
	xiaoyuRuns         *xiaoyuhost.RunManager
	xiaoyuModels       *xiaoyuhost.ModelManager
	xiaoyuIntelligence *xiaoyuhost.IntelligenceStore
	xiaoyuDSHRoot      string
	workspaceFiles     *opsfiles.Service
}

// Options 允许启动器和测试显式注入运行根目录与数据目录。
// 生产环境通常使用 New()；集成测试必须通过 NewWithOptions() 使用独立临时目录，
// 禁止测试回退到真实用户配置目录。
type Options struct {
	Root    string
	DataDir string
}

func New() *Application {
	return NewWithOptions(Options{})
}

func NewWithOptions(options Options) *Application {
	runtimeRoot := strings.TrimSpace(options.Root)
	if runtimeRoot == "" {
		runtimeRoot = platformfiles.ResolveRoot()
	} else {
		runtimeRoot = filepath.Clean(runtimeRoot)
	}
	platformConfig, platformConfigErr := config.NewLoader(runtimeRoot).LoadAll()
	workspaceFiles, workspaceFilesErr := opsfiles.New(runtimeRoot)
	if workspaceFilesErr != nil {
		// runtimeRoot should already be a real project directory; keep construction
		// deterministic for callers and surface the problem through config status.
		platformConfigErr = workspaceFilesErr
	}

	// 运行目录优先读取 configs/paths.json；仅在配置损坏时使用安全回退，确保程序仍能启动并给出诊断。
	dataDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.DataDir, filepath.Join(runtimeRoot, "runtime", "data"))
	if explicitDataDir := strings.TrimSpace(options.DataDir); explicitDataDir != "" {
		dataDir = resolveConfiguredPath(runtimeRoot, explicitDataDir, filepath.Join(runtimeRoot, "runtime", "data"))
	}
	logRoot := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.LogDir, filepath.Join(runtimeRoot, "runtime", "log"))
	instanceDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.InstanceDir, filepath.Join(runtimeRoot, "runtime", "instances"))
	backupDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.BackupDir, filepath.Join(runtimeRoot, "runtime", "backups"))
	tempDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.TempDir, filepath.Join(runtimeRoot, "runtime", "temp"))
	exportDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.ExportDir, filepath.Join(runtimeRoot, "runtime", "exports"))
	pluginDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.PluginDir, filepath.Join(runtimeRoot, "runtime", "plugins"))
	cacheDir := resolveConfiguredPath(runtimeRoot, platformConfig.Paths.CacheDir, filepath.Join(runtimeRoot, "runtime", "cache"))
	logConfig := platformConfig.Logging
	coreLogDir := safeLogSubdir(logConfig.Directories.Core, "agmp")
	operationsLogDir := safeLogSubdir(logConfig.Directories.Operations, "operations")
	gamesLogDir := safeLogSubdir(logConfig.Directories.Games, "games")
	exportLogDir := safeLogSubdir(logConfig.ExportSubdir, "exports")
	logHubService := globallogs.New(logRoot, globallogs.Options{
		ExportSubdir:    exportLogDir,
		CoreDir:         coreLogDir,
		OperationDir:    operationsLogDir,
		AuditDir:        safeLogSubdir(logConfig.Directories.Audit, "audit"),
		AIDir:           safeLogSubdir(logConfig.Directories.AI, "ai"),
		SteamDir:        safeLogSubdir(logConfig.Directories.Steam, "steam"),
		GamesDir:        gamesLogDir,
		NodesDir:        safeLogSubdir(logConfig.Directories.Nodes, "nodes"),
		OperationAudit:  logConfig.OperationAudit,
		FlushIntervalMS: logConfig.FlushIntervalMS,
	})
	steamService := steam.New()
	catalog := []game.CatalogEntry{dst.CatalogEntry()}
	gameWorkspaceService := gameworkspace.New(catalog, steamService)
	dstDedicatedService := dedicated.New(steamService, dstprefs.New(filepath.Join(dataDir, "dst", "settings.json")))
	dstWorkspaceService := dstworkspace.New(gameWorkspaceService, dstDedicatedService)
	dstSetupService := dstsetup.New(dstWorkspaceService)
	// 新版统一放到 log/games/steam.dst；LogHub 仍会索引旧版 log/dst，因此升级不会丢历史日志。
	dstLogStore := logcenter.New(filepath.Join(logRoot, gamesLogDir, string(dst.ID)), filepath.Join(logRoot, exportLogDir))
	dstRuntimeManager := dstruntimecore.NewManager(dstLogStore)
	dstLogService := logcenter.NewService(dstLogStore)
	steamMaintenanceService := steammaintenance.New(steamService, steamprotocol.NewSystemLauncher())
	authService := authservice.New(filepath.Join(dataDir, "auth", "accounts.json"))
	environmentService := environmentservice.New(environmentservice.Options{
		Root: runtimeRoot, DataDir: dataDir, InstanceDir: instanceDir, BackupDir: backupDir,
		TempDir: tempDir, ExportDir: exportDir, PluginDir: pluginDir, CacheDir: cacheDir, Steam: steamService,
	})
	// 许可证公钥是发行信任根，必须编译进二进制，不能从用户可编辑配置读取。
	licenseService := licenseservice.New(licenseservice.DefaultConfig(), filepath.Join(dataDir, "license", "activation.json"))
	xiaoyuRuntimeService := xiaoyuruntime.New(xiaoyuruntime.Options{
		Root:    runtimeRoot,
		Timeout: time.Duration(maxInt(platformConfig.AI.ToolTimeoutSeconds, 30)) * time.Second,
	})
	agentApprovalStore := xiaoyucontrol.New(
		filepath.Join(dataDir, "ai", "approval-state.json"),
		xiaoyucontrol.Mode(strings.TrimSpace(platformConfig.AI.ApprovalMode)),
	)
	updateService := updaterservice.New(updaterservice.Options{
		CurrentVersion: Version,
		CacheDir:       cacheDir,
		Config: updaterservice.Config{
			Enabled: platformConfig.Update.Enabled, Provider: platformConfig.Update.Provider, Repository: platformConfig.Update.Repository,
			APIBaseURL: platformConfig.Update.APIBaseURL, Channel: platformConfig.Update.Channel, CheckOnStartup: platformConfig.Update.CheckOnStartup,
			MinimumCheckIntervalMinutes: platformConfig.Update.MinimumCheckIntervalMinutes, AssetPattern: platformConfig.Update.AssetPattern,
			RequireSHA256: platformConfig.Update.RequireSHA256, InstallMode: platformConfig.Update.InstallMode, PreserveRuntimeData: platformConfig.Update.PreserveRuntimeData,
		},
	})
	xiaoyuTools := xiaoyucontract.New()
	xiaoyuHost := xiaoyuhost.New(xiaoyuTools)
	xiaoyuRuns := xiaoyuhost.NewRunManager(maxInt(platformConfig.AI.ConversationHistoryLimit, 50))
	xiaoyuModels := xiaoyuhost.NewModelManager(filepath.Join(dataDir, "ai", "models.json"), filepath.Join(dataDir, "secrets", "xiaoyu-models"))
	xiaoyuIntelligence := xiaoyuhost.NewIntelligenceStore(filepath.Join(dataDir, "ai", "intelligence.json"))
	app := &Application{
		dataDir:            dataDir,
		logger:             telemetry.New(filepath.Join(logRoot, coreLogDir, "agmp.log")),
		settings:           settings.NewStore(filepath.Join(dataDir, "settings.json")),
		platformConfig:     platformConfig,
		platformConfigErr:  platformConfigErr,
		games:              game.NewRegistry(),
		steam:              steamService,
		gameWorkspace:      gameWorkspaceService,
		dstDedicated:       dstDedicatedService,
		dstWorkspace:       dstWorkspaceService,
		dstRuntime:         dstruntimecore.New(dstWorkspaceService, dstSetupService, dstRuntimeManager),
		dstLogs:            dstLogService,
		dstSetup:           dstSetupService,
		logHub:             logHubService,
		steamMaintenance:   steamMaintenanceService,
		auth:               authService,
		environment:        environmentService,
		license:            licenseService,
		updater:            updateService,
		xiaoyuRuntime:      xiaoyuRuntimeService,
		xiaoyuApprovals:    agentApprovalStore,
		xiaoyuTools:        xiaoyuTools,
		xiaoyuHost:         xiaoyuHost,
		xiaoyuRuns:         xiaoyuRuns,
		xiaoyuModels:       xiaoyuModels,
		xiaoyuIntelligence: xiaoyuIntelligence,
		xiaoyuDSHRoot:      filepath.Join(pluginDir, "dsh"),
		workspaceFiles:     workspaceFiles,
	}
	app.registerXiaoYuTools()
	if err := xiaoyuHost.Add(xiaoyuhost.BrainPlugin{Meta: xiaoyuhost.Manifest{ID: xiaoyuhost.ActiveBrainPluginID, Name: "小鱼模型大脑", Version: Version, Source: "agmp.model-center"}, Provider: &xiaoyuhost.ConfiguredModelBrain{Models: xiaoyuModels, Policy: xiaoyuRuntimeService}}); err != nil {
		app.logger.Warn("register xiaoyu model brain failed", "error", err.Error())
	} else if err := xiaoyuHost.Mount(app.Context(), xiaoyuhost.ActiveBrainPluginID); err != nil {
		app.logger.Warn("mount xiaoyu model brain failed", "error", err.Error())
	}
	return app
}

func maxInt(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func resolveConfiguredPath(root, value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return filepath.Clean(fallback)
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Join(root, filepath.Clean(value))
}

// safeLogSubdir 只接受 log 根目录下的相对目录名，防止配置把日志写出 AI Game Manager Panel 工作区。
func safeLogSubdir(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) {
		return fallback
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return fallback
	}
	return cleaned
}

func resolveDataDir() string {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "AI Game Manager Panel")
	}
	return filepath.Join(os.TempDir(), "AI Game Manager Panel")
}

func (a *Application) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.Info("application startup", "version", Version, "platform", runtime.GOOS)
	if a.xiaoyuRuntime != nil {
		workerCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		if err := a.xiaoyuRuntime.Start(workerCtx); err != nil {
			a.logger.Warn("xiaoyu persistent runtime worker unavailable", "error", err.Error())
		} else {
			a.logger.Info("xiaoyu persistent runtime worker ready")
		}
		cancel()
	}
	a.recordOperation("info", "agmp", "application_start", "AI Game Manager Panel", "success", "应用已启动", "", "")
}

func (a *Application) Shutdown(_ context.Context) {
	if a.xiaoyuRuntime != nil {
		a.xiaoyuRuntime.Close()
	}
	if a.dstRuntime != nil {
		a.logger.Info("application shutdown: stopping managed DST processes")
		a.dstRuntime.StopAllBlocking()
	}
	a.logger.Info("application shutdown")
	a.recordOperation("info", "agmp", "application_stop", "AI Game Manager Panel", "success", "应用已停止", "", "")
	if a.logHub != nil {
		a.logHub.Close()
	}
}
