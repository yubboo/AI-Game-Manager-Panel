package app

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
	updaterservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/updater"
	globallogs "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/logs"
	platformcontract "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/contract"
	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/system/settings"
)

func (a *Application) Ping() string {
	a.logger.Info("backend ping")
	return "AI游戏管理器面板后端连接正常"
}

// Context 仅供桌面 Adapter 使用系统原生对话框等 UI 能力；业务服务不得依赖 Wails。
func (a *Application) Context() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

// PlatformConfig 返回项目根目录 configs/ 的统一配置快照。
// 该接口只暴露非敏感配置；API Key、Token、密码不会出现在这里。
func (a *Application) PlatformConfig() (config.PlatformConfig, error) {
	if a.platformConfigErr != nil {
		return config.PlatformConfig{}, a.platformConfigErr
	}
	return a.platformConfig, nil
}

// PlatformRuntimeContract reports the operating system that actually owns game
// processes. Browser/Desktop clients are control surfaces and never rewrite this
// execution identity.
func (a *Application) PlatformRuntimeContract() platformcontract.RuntimeContract {
	return platformcontract.Current()
}

func (a *Application) Info() Info {
	name := Name
	slogan := Slogan
	if a.platformConfigErr == nil {
		if value := strings.TrimSpace(a.platformConfig.App.ProductName); value != "" {
			name = value
		}
		if value := strings.TrimSpace(a.platformConfig.App.Slogan); value != "" {
			slogan = value
		}
	}

	return Info{
		Name:      name,
		Version:   Version,
		Slogan:    slogan,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		DataDir:   a.dataDir,
	}
}

func (a *Application) Settings() settings.Settings {
	value, err := a.settings.Load()
	if err != nil {
		a.logger.Warn("load settings failed; using defaults", "error", err.Error())
		return settings.Default()
	}
	return value
}

func (a *Application) SaveSettings(value settings.Settings) error {
	if err := a.settings.Save(value); err != nil {
		a.logger.Error("save settings failed", "error", err.Error())
		return err
	}
	a.logger.Info("settings saved", "theme", value.Theme, "debug", value.Debug)
	a.recordOperation("info", "agmp", "save_settings", "平台设置", "success", "保存平台设置", "", "")
	return nil
}

func (a *Application) CheckForUpdates(force bool) (updaterservice.Status, error) {
	ctx := a.Context()
	status, err := a.updater.Check(ctx, force)
	if err != nil {
		a.logger.Warn("update check failed", "error", err.Error())
		return status, err
	}
	if status.UpdateAvailable {
		a.recordOperation("info", "updater", "update_available", status.LatestVersion, "success", status.Message, "", "")
	}
	return status, nil
}

func (a *Application) PrepareLatestUpdate() (updaterservice.PreparedUpdate, error) {
	value, err := a.updater.PrepareLatest(a.Context())
	if err != nil {
		a.logger.Warn("prepare update failed", "error", err.Error())
		return value, err
	}
	a.recordOperation("info", "updater", "download_update", value.Version, "success", "更新安装器已下载并通过 SHA256 校验", "", "")
	return value, nil
}

func (a *Application) LaunchPreparedUpdate(value updaterservice.PreparedUpdate) error {
	if err := a.updater.LaunchInstaller(value); err != nil {
		a.logger.Warn("launch update installer failed", "error", err.Error())
		return err
	}
	a.recordOperation("info", "updater", "launch_installer", value.Version, "success", "已启动 Windows 升级安装器", "", "")
	return nil
}

func (a *Application) RecentLogs(limit int) []string {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return a.logger.ReadRecent(limit)
}

// GlobalLogCatalog 返回 AI Game Manager Panel 全平台日志目录。
func (a *Application) GlobalLogCatalog(request globallogs.CatalogRequest) (globallogs.CatalogPage, error) {
	return a.logHub.Catalog(request)
}

// GlobalLogRead 按需读取一个日志文件，支持从头、从尾、游标继续和内容筛选。
func (a *Application) GlobalLogRead(request globallogs.ReadRequest) (globallogs.ReadPage, error) {
	return a.logHub.Read(request)
}

func (a *Application) ExportGlobalLog(id string) (globallogs.ExportResult, error) {
	value, err := a.logHub.ExportOne(id)
	if err != nil {
		a.logger.Warn("export global log failed", "id", id, "error", err.Error())
		return value, err
	}
	a.recordOperation("info", "logs", "export_log", value.Name, "success", "导出单个日志", "", "")
	return value, nil
}

func (a *Application) ExportGlobalLogs(request globallogs.CatalogRequest) (globallogs.ExportResult, error) {
	value, err := a.logHub.ExportFiltered(request)
	if err != nil {
		a.logger.Warn("export filtered logs failed", "error", err.Error())
		return value, err
	}
	a.recordOperation("info", "logs", "export_logs", value.Name, "success", "导出筛选日志包", "", "")
	return value, nil
}

func (a *Application) DeleteGlobalLog(id string) (globallogs.MutationResult, error) {
	value, err := a.logHub.DeleteOne(id)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "logs", "delete_log", id, "success", "删除单个历史日志", "", "")
	return value, nil
}

func (a *Application) DeleteGlobalLogs(request globallogs.DeleteFilteredRequest) (globallogs.MutationResult, error) {
	value, err := a.logHub.DeleteFiltered(request)
	if err == nil {
		a.recordOperation("warning", "logs", "delete_filtered_logs", "日志中心", "success", fmt.Sprintf("删除 %d 个，跳过 %d 个活动日志", value.Deleted, value.Skipped), "", "")
	}
	return value, err
}

func (a *Application) ClearGlobalLogHistory() (globallogs.MutationResult, error) {
	value, err := a.logHub.ClearHistory()
	if err == nil {
		a.recordOperation("warning", "logs", "clear_log_history", "日志中心", "success", fmt.Sprintf("删除 %d 个，跳过 %d 个活动日志", value.Deleted, value.Skipped), "", "")
	}
	return value, err
}

func (a *Application) OpenGlobalLogFolder() error {
	if err := platformfiles.OpenFolder(a.logHub.Root()); err != nil {
		return err
	}
	a.recordOperation("info", "logs", "open_log_folder", "log", "success", "打开日志目录", "", "")
	return nil
}

func (a *Application) recordOperation(level, source, action, target, result, detail, gameID, instance string) {
	if a.logHub == nil {
		return
	}
	a.logHub.RecordOperation(globallogs.OperationRecord{
		Level: level, Source: source, Action: action, Target: target, Result: result, Detail: detail, GameID: gameID, Instance: instance,
	})
}

func (a *Application) CurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// SteamEnvironment discovers the shared Steam platform on the current node.
// Steam not being installed is a normal state and is returned as Detected=false.
