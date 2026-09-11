package app

import (
	"context"
	"time"

	environmentservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
)

func (a *Application) EnvironmentSetupStatus() environmentservice.Status {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return a.environment.Status(ctx)
}

func (a *Application) InitializeEnvironment(request environmentservice.InitializeRequest) (environmentservice.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	status, err := a.environment.Initialize(ctx, request)
	if err != nil {
		a.logger.Warn("environment initialization failed", "error", err.Error())
		return status, err
	}
	a.recordOperation("info", "environment", "initialize", status.DefaultInstallRoot, "success", "运行环境初始化完成", "", "")
	return status, nil
}

func (a *Application) SkipEnvironmentSetup() (environmentservice.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	status, err := a.environment.Skip(ctx)
	if err != nil {
		a.logger.Warn("environment setup skip failed", "error", err.Error())
		return status, err
	}
	a.recordOperation("info", "environment", "skip_initial_setup", status.DefaultInstallRoot, "success", "用户选择暂时跳过运行环境初始化", "", "")
	return status, nil
}

func (a *Application) InstallSteamCMD() (environmentservice.Status, error) {
	return a.InstallSteamCMDAt(environmentservice.SteamCMDInstallRequest{})
}

func (a *Application) InstallSteamCMDAt(request environmentservice.SteamCMDInstallRequest) (environmentservice.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	status, err := a.environment.InstallSteamCMDAt(ctx, request)
	if err != nil {
		a.logger.Warn("steamcmd installation failed", "error", err.Error())
		return status, err
	}
	a.recordOperation("info", "environment", "install_steamcmd", status.SteamCMD.Path, "success", "SteamCMD 已安装", "", "")
	return status, nil
}

func (a *Application) UpdateEnvironmentPaths(request environmentservice.StoragePathsRequest) (environmentservice.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	status, err := a.environment.UpdateStoragePaths(ctx, request)
	if err != nil {
		a.logger.Warn("environment paths update failed", "error", err.Error())
		return status, err
	}
	a.recordOperation("info", "environment", "update_paths", status.Paths.GameLibraryRoot, "success", "运行环境与存储路径已更新", "", "")
	return status, nil
}

func (a *Application) MigrateSteamCMD(request environmentservice.MigratePathRequest) (environmentservice.MigrationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := a.environment.MigrateSteamCMD(ctx, request)
	if err != nil {
		a.logger.Warn("steamcmd migration failed", "error", err.Error())
		return result, err
	}
	a.recordOperation("info", "environment", "migrate_steamcmd", result.Target, "success", result.Message, "", "")
	return result, nil
}

func (a *Application) MigrateGameLibrary(request environmentservice.MigratePathRequest) (environmentservice.MigrationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	result, err := a.environment.MigrateGameLibrary(ctx, request)
	if err != nil {
		a.logger.Warn("game library migration failed", "error", err.Error())
		return result, err
	}
	a.recordOperation("info", "environment", "migrate_game_library", result.Target, "success", result.Message, "", "")
	return result, nil
}

func (a *Application) EnvironmentRuntimeCatalog() environmentservice.RuntimeCatalog {
	return a.environment.RuntimeCatalog()
}

func (a *Application) InstallJavaRuntime(request environmentservice.JavaInstallRequest) (environmentservice.RuntimeRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	value, err := a.environment.InstallJava(ctx, request)
	if err != nil {
		a.logger.Warn("java runtime installation failed", "major", request.Major, "error", err.Error())
		return value, err
	}
	a.recordOperation("info", "environment", "install_java", value.Executable, "success", "Java Runtime 已安装并验证", "", "")
	return value, nil
}

func (a *Application) RegisterEnvironmentRuntime(request environmentservice.RegisterRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	value, err := a.environment.RegisterRuntime(request)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "environment", "register_runtime", value.Executable, "success", "外部 Runtime 已登记并验证", "", "")
	return value, nil
}

func (a *Application) ResolveEnvironmentRuntime(request environmentservice.ResolveRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	return a.environment.ResolveRuntime(request)
}

func (a *Application) SetDefaultEnvironmentRuntime(request environmentservice.SetDefaultRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	value, err := a.environment.SetDefaultRuntime(request)
	if err != nil {
		return value, err
	}
	a.recordOperation("info", "environment", "set_default_runtime", value.ID, "success", "默认 Runtime 已更新", "", "")
	return value, nil
}

func (a *Application) RemoveEnvironmentRuntime(request environmentservice.RemoveRuntimeRequest) error {
	if err := a.environment.RemoveRuntime(request); err != nil {
		return err
	}
	a.recordOperation("warn", "environment", "remove_runtime", request.ID, "success", "Runtime 记录已移除", "", "")
	return nil
}

func (a *Application) EnvironmentGameRuntimeProfiles() []environmentservice.GameRuntimeProfile {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return a.environment.GameRuntimeProfiles(ctx)
}

func (a *Application) EnvironmentGameRuntimeProfile(request environmentservice.GameRuntimeProfileRequest) environmentservice.GameRuntimeProfile {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return a.environment.GameRuntimeProfile(ctx, request)
}

func (a *Application) InstallEnvironmentSystemPrerequisite(request environmentservice.InstallSystemPrerequisiteRequest) (environmentservice.SystemPrerequisite, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	value, err := a.environment.InstallSystemPrerequisite(ctx, request)
	if err != nil {
		return value, err
	}
	a.recordOperation("warn", "environment", "install_system_prerequisite", request.ID, "success", "Linux 系统 Runtime 前置依赖已安装", "", "")
	return value, nil
}
