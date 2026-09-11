package wailsbridge

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
	environmentservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	steammaintenance "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/steam/maintenance"
	updaterservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/updater"
	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/logcenter"
	dstprefs "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	dstruntimecore "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/workspace"
	clusterops "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/cluster"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/kleiarchive"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/preflight"
	dsttoken "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst/token"
	globallogs "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/logs"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/server/workspace"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	licenseservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/system/settings"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
	"time"
)

// App is a thin desktop adapter. It contains no game or filesystem business logic.
type App struct {
	application *application.Application
}

func New(application *application.Application) *App {
	return &App{application: application}
}

// requireSession is the desktop equivalent of the HTTP /api session middleware.
// Wails methods do not pass through net/http, so every organization-owned
// resource must explicitly validate the current AGMP member session here.
func (a *App) requireSession(token string) error {
	_, err := a.application.ValidateSession(token)
	return err
}

func (a *App) requireAdministrator(token string) error {
	_, err := a.application.RequireOrganizationAdministrator(token)
	return err
}

func (a *App) Ping() string {
	return a.application.Ping()
}

func (a *App) CheckForUpdates(token string, force bool) (updaterservice.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero updaterservice.Status
		return zero, err
	}
	return a.application.CheckForUpdates(force)
}

func (a *App) InstallLatestUpdate(token string) (updaterservice.PreparedUpdate, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero updaterservice.PreparedUpdate
		return zero, err
	}
	value, err := a.application.PrepareLatestUpdate()
	if err != nil {
		return value, err
	}
	if err := a.application.LaunchPreparedUpdate(value); err != nil {
		return value, err
	}
	ctx := a.application.Context()
	go func() {
		time.Sleep(700 * time.Millisecond)
		wailsruntime.Quit(ctx)
	}()
	return value, nil
}

func (a *App) SelectDirectory(token string, title string, initial string) (string, error) {
	if err := a.requireSession(token); err != nil {
		var zero string
		return zero, err
	}
	return wailsruntime.OpenDirectoryDialog(a.application.Context(), wailsruntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: initial,
	})
}

func (a *App) GetLicenseStatus(token string) (licenseservice.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero licenseservice.Status
		return zero, err
	}
	return a.application.LicenseStatus(), nil
}

func (a *App) ActivateLicense(token string, request licenseservice.ActivateRequest) (licenseservice.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero licenseservice.Status
		return zero, err
	}
	return a.application.ActivateLicenseForUser(token, request)
}

func (a *App) UnbindLicense(token string) (licenseservice.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero licenseservice.Status
		return zero, err
	}
	return a.application.UnbindLicenseForUser(token)
}

func (a *App) GetLicenseFeatureStatus(token string, feature string) (licenseservice.FeatureStatus, error) {
	if err := a.requireSession(token); err != nil {
		var zero licenseservice.FeatureStatus
		return zero, err
	}
	return a.application.LicenseFeatureStatus(feature), nil
}

func (a *App) GetXiaoYuRuntimeStatus(token string) (xiaoyuruntime.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero xiaoyuruntime.Status
		return zero, err
	}
	return a.application.XiaoYuRuntimeStatus(), nil
}

func (a *App) GetXiaoYuTools(token string) ([]xiaoyuruntime.ToolSpec, error) {
	return a.application.XiaoYuTools(token)
}

func (a *App) GetXiaoYuCapabilities(token string) (xiaoyuhost.CapabilitySnapshot, error) {
	return a.application.XiaoYuCapabilities(token)
}

func (a *App) GetXiaoYuHarnessStatus(token string) (application.XiaoYuHarnessSnapshot, error) {
	return a.application.XiaoYuHarnessStatus(token)
}

func (a *App) GetXiaoYuModelCatalog(token string) (xiaoyuhost.ModelCatalog, error) {
	return a.application.XiaoYuModelCatalog(token)
}

func (a *App) SaveXiaoYuModel(token string, request xiaoyuhost.SaveModelRequest) (xiaoyuhost.ModelProfileView, error) {
	return a.application.SaveXiaoYuModel(token, request)
}

func (a *App) DeleteXiaoYuModel(token, id string) error {
	return a.application.DeleteXiaoYuModel(token, id)
}

func (a *App) SetXiaoYuDefaultModel(token, id string) (xiaoyuhost.ModelProfileView, error) {
	return a.application.SetXiaoYuDefaultModel(token, id)
}

func (a *App) TestXiaoYuModel(token string, request xiaoyuhost.ModelConnectionRequest) (xiaoyuhost.ModelConnectionResult, error) {
	return a.application.TestXiaoYuModel(token, request)
}

func (a *App) DiscoverXiaoYuModels(token string, request xiaoyuhost.ModelConnectionRequest) (xiaoyuhost.ModelConnectionResult, error) {
	return a.application.DiscoverXiaoYuModels(token, request)
}

func (a *App) GetXiaoYuIntelligenceCatalog(token string) (xiaoyuhost.IntelligenceCatalog, error) {
	return a.application.XiaoYuIntelligenceCatalog(token)
}

func (a *App) SaveXiaoYuMemory(token string, request xiaoyuhost.MemorySaveRequest) (xiaoyuhost.MemoryRecord, error) {
	return a.application.SaveXiaoYuMemory(token, request)
}

func (a *App) SaveXiaoYuSkill(token string, request xiaoyuhost.SkillSaveRequest) (xiaoyuhost.SkillDefinition, error) {
	return a.application.SaveXiaoYuSkill(token, request)
}

func (a *App) SaveXiaoYuExpert(token string, request xiaoyuhost.ExpertSaveRequest) (xiaoyuhost.ExpertDefinition, error) {
	return a.application.SaveXiaoYuExpert(token, request)
}

func (a *App) DeleteXiaoYuIntelligence(token, kind, id string) error {
	return a.application.DeleteXiaoYuIntelligence(token, kind, id)
}

func (a *App) StartXiaoYuRun(token string, request application.XiaoYuRunRequest) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuStartRun(token, request)
}

func (a *App) ContinueXiaoYuRun(token, id string, request application.XiaoYuContinueRunRequest) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuContinueRun(token, id, request)
}

func (a *App) GetXiaoYuRunState(token, id string) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuRunState(token, id)
}

func (a *App) CancelXiaoYuRun(token, id string) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuCancelRun(token, id)
}

func (a *App) PauseXiaoYuRun(token, id string, request application.XiaoYuControlRequest) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuPauseRun(token, id, request)
}

func (a *App) TakeoverXiaoYuRun(token, id string, request application.XiaoYuControlRequest) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuTakeoverRun(token, id, request)
}

func (a *App) ResumeXiaoYuRun(token, id string) (xiaoyuhost.RunState, error) {
	return a.application.XiaoYuResumeRun(token, id)
}

func (a *App) GetXiaoYuTrace(token string, after uint64, limit int) ([]xiaoyuhost.Event, error) {
	return a.application.XiaoYuTrace(token, after, limit)
}

func (a *App) GetXiaoYuDSHPlugins(token string) ([]xiaoyuhost.DSHBundle, error) {
	return a.application.XiaoYuDSHPlugins(token)
}

func (a *App) MountXiaoYuDSHPlugin(token string, request xiaoyuhost.DSHMountRequest) (xiaoyuhost.PluginSnapshot, error) {
	return a.application.MountXiaoYuDSHPlugin(token, request)
}

func (a *App) UnmountXiaoYuPlugin(token, id string) error {
	return a.application.UnmountXiaoYuPlugin(token, id)
}

func (a *App) GetXiaoYuApprovalState(token string) (xiaoyucontrol.Snapshot, error) {
	return a.application.XiaoYuApprovalState(token)
}

func (a *App) SetXiaoYuApprovalMode(token string, request xiaoyucontrol.SetModeRequest) (xiaoyucontrol.Snapshot, error) {
	return a.application.SetXiaoYuApprovalMode(token, request)
}

func (a *App) GetXiaoYuPendingApprovals(token string) ([]xiaoyucontrol.Request, error) {
	return a.application.XiaoYuPendingApprovals(token)
}

func (a *App) ResolveXiaoYuApproval(token, id string, request xiaoyucontrol.ResolveRequest) (xiaoyucontrol.Request, error) {
	return a.application.ResolveXiaoYuApproval(token, id, request)
}

func (a *App) CallXiaoYuTool(token string, request xiaoyuruntime.ToolCallRequest) (xiaoyuruntime.ToolCallResult, error) {
	return a.application.XiaoYuCallTool(token, request)
}

func (a *App) RunXiaoYuCommand(token string, request xiaoyuruntime.CommandRequest) (xiaoyuruntime.CommandResult, error) {
	return a.application.XiaoYuRunCommand(token, request)
}

func (a *App) GetAuthBootstrapStatus() authservice.BootstrapStatus {
	return a.application.AuthBootstrapStatus()
}

func (a *App) GenerateSecurityKey() (authservice.SecurityKeyMaterial, error) {
	return a.application.GenerateSecurityKey()
}

func (a *App) CreateInitialAdministrator(request authservice.CreateOwnerRequest) (authservice.Session, error) {
	return a.application.CreateInitialAdministrator(request)
}

func (a *App) Login(request authservice.LoginRequest) (authservice.Session, error) {
	return a.application.Login(request)
}

func (a *App) ValidateSession(token string) (authservice.User, error) {
	return a.application.ValidateSession(token)
}

func (a *App) Logout(token string) {
	a.application.Logout(token)
}

func (a *App) ListUsers(token string) ([]authservice.User, error) {
	return a.application.ListUsers(token)
}

func (a *App) UpdateMyDisplayName(token string, request authservice.UpdateDisplayNameRequest) (authservice.User, error) {
	return a.application.UpdateMyDisplayName(token, request)
}

func (a *App) GetAuthInstanceIdentity() authservice.InstanceIdentity {
	return a.application.AuthInstanceIdentity()
}

func (a *App) GetAuthOrganization(token string) (authservice.Organization, error) {
	return a.application.AuthOrganization(token)
}

func (a *App) InspectUserInvitation(token string) (authservice.InvitationPreview, error) {
	return a.application.InspectUserInvitation(token)
}

func (a *App) RegisterInvitedUser(request authservice.RegisterInvitationRequest) (authservice.Session, error) {
	return a.application.RegisterInvitedUser(request)
}

func (a *App) CreateUserInvitation(token string, request authservice.CreateInvitationRequest) (authservice.CreatedInvitation, error) {
	return a.application.CreateUserInvitation(token, request)
}

func (a *App) ListUserInvitations(token string) ([]authservice.InvitationView, error) {
	return a.application.ListUserInvitations(token)
}

func (a *App) RevokeUserInvitation(token, id string) (authservice.InvitationView, error) {
	return a.application.RevokeUserInvitation(token, id)
}

func (a *App) GetMyEmailSecurityStatus(token string) (authservice.EmailSecurityStatus, error) {
	return a.application.MyEmailSecurityStatus(token)
}

func (a *App) BindMyEmail(token string, request authservice.BindEmailRequest) (authservice.EmailSecurityStatus, error) {
	return a.application.BindMyEmail(token, request)
}

func (a *App) UnbindMyEmail(token, password string) (authservice.EmailSecurityStatus, error) {
	return a.application.UnbindMyEmail(token, password)
}

func (a *App) RequestMyEmailVerification(token string, request authservice.RequestEmailVerificationRequest) error {
	return a.application.RequestMyEmailVerification(token, request)
}

func (a *App) ConfirmMyEmailVerification(token string, request authservice.ConfirmEmailVerificationRequest) (authservice.EmailSecurityStatus, error) {
	return a.application.ConfirmMyEmailVerification(token, request)
}

func (a *App) ConfirmMyCredentialStepUp(token string, request authservice.CredentialStepUpRequest) (authservice.User, error) {
	return a.application.ConfirmMyCredentialStepUp(token, request)
}

func (a *App) RequestPasswordReset(request authservice.RequestPasswordResetRequest) (authservice.PasswordResetRequestStatus, error) {
	return a.application.RequestPasswordReset(request)
}

func (a *App) ConfirmPasswordReset(request authservice.ConfirmPasswordResetRequest) error {
	return a.application.ConfirmPasswordReset(request)
}

func (a *App) GetSMTPSettings(token string) (authservice.SMTPSettings, error) {
	return a.application.SMTPSettings(token)
}

func (a *App) SaveSMTPSettings(token string, request authservice.SaveSMTPSettingsRequest) (authservice.SMTPSettings, error) {
	return a.application.SaveSMTPSettings(token, request)
}

func (a *App) CreateMemberCoreAuthorization(token string, request authservice.CreateMemberAuthorizationRequest) (authservice.CreatedMemberAuthorization, error) {
	return a.application.CreateMemberCoreAuthorization(token, request)
}

func (a *App) ListMemberCoreAuthorizations(token string) ([]authservice.MemberAuthorizationView, error) {
	return a.application.ListMemberCoreAuthorizations(token)
}

func (a *App) RevokeMemberCoreAuthorization(token, id string) (authservice.MemberAuthorizationView, error) {
	return a.application.RevokeMemberCoreAuthorization(token, id)
}

func (a *App) RedeemMyCoreAuthorization(token string, request authservice.RedeemMemberAuthorizationRequest) (authservice.User, error) {
	return a.application.RedeemMyCoreAuthorization(token, request)
}

func (a *App) RevokeMemberCoreAccess(token, userID string) (authservice.User, error) {
	return a.application.RevokeMemberCoreAccess(token, userID)
}

func (a *App) ClearMemberRisk(token, userID string) (authservice.User, error) {
	return a.application.ClearMemberRisk(token, userID)
}

func (a *App) RemoveOrganizationMember(token, userID string) error {
	return a.application.RemoveOrganizationMember(token, userID)
}

func (a *App) GetMySecurityStatus(token string) (authservice.SecurityStatus, error) {
	return a.application.MySecurityStatus(token)
}

func (a *App) RotateMySecurityKey(token string, request authservice.RotateSecurityKeyRequest) (authservice.SecurityKeyMaterial, error) {
	return a.application.RotateMySecurityKey(token, request)
}

func (a *App) SetMySecurityKeyVerification(token string, request authservice.SetSecurityKeyVerificationRequest) (authservice.SecurityStatus, error) {
	return a.application.SetMySecurityKeyVerification(token, request)
}

func (a *App) GetEnvironmentSetupStatus(token string) (environmentservice.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.EnvironmentSetupStatus(), nil
}

func (a *App) InitializeEnvironment(token string, request environmentservice.InitializeRequest) (environmentservice.Status, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.InitializeEnvironment(request)
}

func (a *App) SkipEnvironmentSetup(token string) (environmentservice.Status, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.SkipEnvironmentSetup()
}

func (a *App) InstallSteamCMD(token string) (environmentservice.Status, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.InstallSteamCMD()
}

func (a *App) InstallSteamCMDAt(token string, request environmentservice.SteamCMDInstallRequest) (environmentservice.Status, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.InstallSteamCMDAt(request)
}

func (a *App) UpdateEnvironmentPaths(token string, request environmentservice.StoragePathsRequest) (environmentservice.Status, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.Status
		return zero, err
	}
	return a.application.UpdateEnvironmentPaths(request)
}

func (a *App) MigrateSteamCMD(token string, request environmentservice.MigratePathRequest) (environmentservice.MigrationResult, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.MigrationResult
		return zero, err
	}
	return a.application.MigrateSteamCMD(request)
}

func (a *App) MigrateGameLibrary(token string, request environmentservice.MigratePathRequest) (environmentservice.MigrationResult, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero environmentservice.MigrationResult
		return zero, err
	}
	return a.application.MigrateGameLibrary(request)
}

func (a *App) GetEnvironmentRuntimeCatalog(token string) (environmentservice.RuntimeCatalog, error) {
	if err := a.requireSession(token); err != nil {
		return environmentservice.RuntimeCatalog{}, err
	}
	return a.application.EnvironmentRuntimeCatalog(), nil
}

func (a *App) InstallJavaRuntime(token string, request environmentservice.JavaInstallRequest) (environmentservice.RuntimeRecord, error) {
	if err := a.requireAdministrator(token); err != nil {
		return environmentservice.RuntimeRecord{}, err
	}
	return a.application.InstallJavaRuntime(request)
}

func (a *App) RegisterEnvironmentRuntime(token string, request environmentservice.RegisterRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	if err := a.requireAdministrator(token); err != nil {
		return environmentservice.RuntimeRecord{}, err
	}
	return a.application.RegisterEnvironmentRuntime(request)
}

func (a *App) ResolveEnvironmentRuntime(token string, request environmentservice.ResolveRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	if err := a.requireSession(token); err != nil {
		return environmentservice.RuntimeRecord{}, err
	}
	return a.application.ResolveEnvironmentRuntime(request)
}

func (a *App) SetDefaultEnvironmentRuntime(token string, request environmentservice.SetDefaultRuntimeRequest) (environmentservice.RuntimeRecord, error) {
	if err := a.requireAdministrator(token); err != nil {
		return environmentservice.RuntimeRecord{}, err
	}
	return a.application.SetDefaultEnvironmentRuntime(request)
}

func (a *App) RemoveEnvironmentRuntime(token string, request environmentservice.RemoveRuntimeRequest) error {
	if err := a.requireAdministrator(token); err != nil {
		return err
	}
	return a.application.RemoveEnvironmentRuntime(request)
}

func (a *App) GetEnvironmentGameRuntimeProfiles(token string) ([]environmentservice.GameRuntimeProfile, error) {
	if err := a.requireSession(token); err != nil {
		return nil, err
	}
	return a.application.EnvironmentGameRuntimeProfiles(), nil
}

func (a *App) GetEnvironmentGameRuntimeProfile(token string, request environmentservice.GameRuntimeProfileRequest) (environmentservice.GameRuntimeProfile, error) {
	if err := a.requireSession(token); err != nil {
		return environmentservice.GameRuntimeProfile{}, err
	}
	return a.application.EnvironmentGameRuntimeProfile(request), nil
}

func (a *App) InstallEnvironmentSystemPrerequisite(token string, request environmentservice.InstallSystemPrerequisiteRequest) (environmentservice.SystemPrerequisite, error) {
	if err := a.requireAdministrator(token); err != nil {
		return environmentservice.SystemPrerequisite{}, err
	}
	return a.application.InstallEnvironmentSystemPrerequisite(request)
}

func (a *App) GetAppInfo(token string) (application.Info, error) {
	if err := a.requireSession(token); err != nil {
		var zero application.Info
		return zero, err
	}
	return a.application.Info(), nil
}

func (a *App) GetPlatformConfig(token string) (config.PlatformConfig, error) {
	if err := a.requireSession(token); err != nil {
		var zero config.PlatformConfig
		return zero, err
	}
	return a.application.PlatformConfig()
}

func (a *App) GetSettings(token string) (settings.Settings, error) {
	if err := a.requireSession(token); err != nil {
		var zero settings.Settings
		return zero, err
	}
	return a.application.Settings(), nil
}

func (a *App) SaveSettings(token string, value settings.Settings) error {
	if err := a.requireAdministrator(token); err != nil {
		return err
	}
	return a.application.SaveSettings(value)
}

func (a *App) GetRecentLogs(token string, limit int) ([]string, error) {
	if err := a.requireSession(token); err != nil {
		var zero []string
		return zero, err
	}
	return a.application.RecentLogs(limit), nil
}

func (a *App) GetGlobalLogCatalog(token string, request globallogs.CatalogRequest) (globallogs.CatalogPage, error) {
	if err := a.requireSession(token); err != nil {
		var zero globallogs.CatalogPage
		return zero, err
	}
	return a.application.GlobalLogCatalog(request)
}

func (a *App) ReadGlobalLog(token string, request globallogs.ReadRequest) (globallogs.ReadPage, error) {
	if err := a.requireSession(token); err != nil {
		var zero globallogs.ReadPage
		return zero, err
	}
	return a.application.GlobalLogRead(request)
}

func (a *App) ExportGlobalLog(token string, id string) (globallogs.ExportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero globallogs.ExportResult
		return zero, err
	}
	return a.application.ExportGlobalLog(id)
}

func (a *App) ExportGlobalLogs(token string, request globallogs.CatalogRequest) (globallogs.ExportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero globallogs.ExportResult
		return zero, err
	}
	return a.application.ExportGlobalLogs(request)
}

func (a *App) DeleteGlobalLog(token string, id string) (globallogs.MutationResult, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero globallogs.MutationResult
		return zero, err
	}
	return a.application.DeleteGlobalLog(id)
}

func (a *App) DeleteGlobalLogs(token string, request globallogs.DeleteFilteredRequest) (globallogs.MutationResult, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero globallogs.MutationResult
		return zero, err
	}
	return a.application.DeleteGlobalLogs(request)
}

func (a *App) ClearGlobalLogHistory(token string) (globallogs.MutationResult, error) {
	if err := a.requireAdministrator(token); err != nil {
		var zero globallogs.MutationResult
		return zero, err
	}
	return a.application.ClearGlobalLogHistory()
}

func (a *App) OpenGlobalLogFolder(token string) error {
	if err := a.requireAdministrator(token); err != nil {
		return err
	}
	return a.application.OpenGlobalLogFolder()
}

func (a *App) CurrentTime(token string) (string, error) {
	if err := a.requireSession(token); err != nil {
		var zero string
		return zero, err
	}
	return a.application.CurrentTime(), nil
}

func (a *App) GetSteamEnvironment(token string) (steam.Environment, error) {
	if err := a.requireSession(token); err != nil {
		var zero steam.Environment
		return zero, err
	}
	return a.application.SteamEnvironment()
}

func (a *App) GetSteamApps(token string) (steam.AppInventory, error) {
	if err := a.requireSession(token); err != nil {
		var zero steam.AppInventory
		return zero, err
	}
	return a.application.SteamApps()
}

func (a *App) GetSteamSnapshot(token string) (steam.Snapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero steam.Snapshot
		return zero, err
	}
	return a.application.SteamSnapshot()
}
func (a *App) GetGameWorkspaceSnapshot(token string, id string) (gameworkspace.Snapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero gameworkspace.Snapshot
		return zero, err
	}
	return a.application.GameWorkspaceSnapshot(id)
}

func (a *App) GetDSTEnvironment(token string) (dstdomain.Environment, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstdomain.Environment
		return zero, err
	}
	return a.application.DSTEnvironment(), nil
}

func (a *App) GetDSTWorkspaceSnapshot(token string) (dstworkspace.Snapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstworkspace.Snapshot
		return zero, err
	}
	return a.application.DSTWorkspaceSnapshot()
}

func (a *App) GetDSTDedicatedServer(token string) (dedicated.Snapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dedicated.Snapshot
		return zero, err
	}
	return a.application.DSTDedicatedServer()
}

func (a *App) GetDSTDedicatedPreferences(token string) (dstprefs.Value, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstprefs.Value
		return zero, err
	}
	return a.application.DSTDedicatedPreferences(), nil
}

func (a *App) SaveDSTDedicatedPreferences(token string, value dstprefs.Value) error {
	if err := a.requireSession(token); err != nil {
		return err
	}
	return a.application.SaveDSTDedicatedPreferences(value)
}

func (a *App) OpenDSTTokenPage(token string) error {
	if err := a.requireSession(token); err != nil {
		return err
	}
	return a.application.OpenDSTTokenPage()
}

func (a *App) GetDSTTokenStatus(token string, clusterPath string) (dsttoken.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero dsttoken.Status
		return zero, err
	}
	return a.application.DSTTokenStatus(clusterPath)
}

func (a *App) SaveDSTToken(token string, request dsttoken.SaveRequest) (dsttoken.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero dsttoken.Status
		return zero, err
	}
	return a.application.SaveDSTToken(request)
}

func (a *App) ImportDSTToken(token string, request dsttoken.ImportRequest) (dsttoken.Status, error) {
	if err := a.requireSession(token); err != nil {
		var zero dsttoken.Status
		return zero, err
	}
	return a.application.ImportDSTToken(request)
}

func (a *App) InspectDSTKleiPackage(token string, request kleiarchive.InspectRequest) (kleiarchive.Preview, error) {
	if err := a.requireSession(token); err != nil {
		var zero kleiarchive.Preview
		return zero, err
	}
	return a.application.InspectDSTKleiPackage(request)
}

func (a *App) ImportDSTKleiPackage(token string, request kleiarchive.ImportRequest) (kleiarchive.ImportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero kleiarchive.ImportResult
		return zero, err
	}
	return a.application.ImportDSTKleiPackage(request)
}

func (a *App) GetDSTPreflight(token string, clusterPath string) (preflight.Result, error) {
	if err := a.requireSession(token); err != nil {
		var zero preflight.Result
		return zero, err
	}
	return a.application.DSTPreflight(clusterPath)
}

func (a *App) ImportDSTCluster(token string, request clusterops.ImportRequest) (clusterops.ImportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero clusterops.ImportResult
		return zero, err
	}
	return a.application.ImportDSTCluster(request)
}

func (a *App) BuildDSTLaunchSpec(token string, request dedicated.LaunchRequest) (dedicated.LaunchSpec, error) {
	if err := a.requireSession(token); err != nil {
		var zero dedicated.LaunchSpec
		return zero, err
	}
	return a.application.BuildDSTLaunchSpec(request)
}

func (a *App) StartDSTMaster(token string, request dstruntimecore.StartMasterRequest) (dstruntimecore.ProcessSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ProcessSnapshot
		return zero, err
	}
	return a.application.StartDSTMaster(request)
}

func (a *App) StartDSTCluster(token string, request dstruntimecore.StartClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ClusterRuntimeSnapshot
		return zero, err
	}
	return a.application.StartDSTCluster(request)
}

func (a *App) StartDSTShard(token string, request dstruntimecore.StartShardRequest) (dstruntimecore.ProcessSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ProcessSnapshot
		return zero, err
	}
	return a.application.StartDSTShard(request)
}

func (a *App) GetDSTClusterStatus(token string, request dstruntimecore.ClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ClusterRuntimeSnapshot
		return zero, err
	}
	return a.application.DSTClusterStatus(request)
}

func (a *App) GetDSTPortStatus(token string, request dstruntimecore.ClusterRequest) (dstruntimecore.PortReport, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.PortReport
		return zero, err
	}
	return a.application.DSTPortStatus(request)
}

func (a *App) GetDSTPortConfiguration(token string, request dstruntimecore.ClusterRequest) (dedicated.PortConfiguration, error) {
	if err := a.requireSession(token); err != nil {
		var zero dedicated.PortConfiguration
		return zero, err
	}
	return a.application.DSTPortConfiguration(request)
}

func (a *App) ConfigureDSTPorts(token string, request dstruntimecore.PortConfigureRequest) (dstruntimecore.PortConfigureResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.PortConfigureResult
		return zero, err
	}
	return a.application.ConfigureDSTPorts(request)
}

func (a *App) CleanupDSTPorts(token string, request dstruntimecore.PortCleanupRequest) (dstruntimecore.PortCleanupResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.PortCleanupResult
		return zero, err
	}
	return a.application.CleanupDSTPorts(request)
}

func (a *App) StopDSTCluster(token string, request dstruntimecore.ClusterRequest) (dstruntimecore.ClusterRuntimeSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ClusterRuntimeSnapshot
		return zero, err
	}
	return a.application.StopDSTCluster(request)
}

func (a *App) GetDSTProcessStatus(token string, request dstruntimecore.ProcessRequest) (dstruntimecore.LookupResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.LookupResult
		return zero, err
	}
	return a.application.DSTProcessStatus(request), nil
}

func (a *App) ReadDSTProcessLogs(token string, request dstruntimecore.LogRequest) (dstruntimecore.LogBatch, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.LogBatch
		return zero, err
	}
	return a.application.DSTProcessLogs(request)
}

func (a *App) SendDSTCommand(token string, request dstruntimecore.CommandRequest) (dstruntimecore.ProcessSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ProcessSnapshot
		return zero, err
	}
	return a.application.SendDSTCommand(request)
}

func (a *App) StopDSTProcess(token string, request dstruntimecore.ProcessRequest) (dstruntimecore.ProcessSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero dstruntimecore.ProcessSnapshot
		return zero, err
	}
	return a.application.StopDSTProcess(request)
}

func (a *App) GetDSTLogSessions(token string, request logcenter.ListRequest) ([]logcenter.Session, error) {
	if err := a.requireSession(token); err != nil {
		var zero []logcenter.Session
		return zero, err
	}
	return a.application.DSTLogSessions(request)
}

func (a *App) ReadDSTLog(token string, request logcenter.ReadRequest) (logcenter.ReadPage, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.ReadPage
		return zero, err
	}
	return a.application.DSTLogRead(request)
}

func (a *App) TailDSTLog(token string, request logcenter.TailRequest) (logcenter.ReadPage, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.ReadPage
		return zero, err
	}
	return a.application.DSTLogTail(request)
}

func (a *App) SearchDSTLog(token string, request logcenter.SearchRequest) (logcenter.SearchResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.SearchResult
		return zero, err
	}
	return a.application.DSTLogSearch(request)
}

func (a *App) GetDSTLogDiagnostics(token string, id string) (logcenter.Diagnostics, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.Diagnostics
		return zero, err
	}
	return a.application.DSTLogDiagnostics(id)
}

func (a *App) ExportDSTLog(token string, id string) (logcenter.ExportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.ExportResult
		return zero, err
	}
	return a.application.ExportDSTLog(id)
}

func (a *App) ExportDSTLogBundle(token string, request logcenter.BundleRequest) (logcenter.ExportResult, error) {
	if err := a.requireSession(token); err != nil {
		var zero logcenter.ExportResult
		return zero, err
	}
	return a.application.ExportDSTLogBundle(request)
}

func (a *App) StartSteamInstall(token string, request steammaintenance.InstallRequest) (steammaintenance.TaskSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero steammaintenance.TaskSnapshot
		return zero, err
	}
	return a.application.StartSteamInstall(request)
}

func (a *App) StartSteamValidation(token string, request steammaintenance.ValidateRequest) (steammaintenance.TaskSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero steammaintenance.TaskSnapshot
		return zero, err
	}
	return a.application.StartSteamValidation(request)
}

func (a *App) GetSteamMaintenanceTask(token string, id string) (steammaintenance.TaskSnapshot, error) {
	if err := a.requireSession(token); err != nil {
		var zero steammaintenance.TaskSnapshot
		return zero, err
	}
	return a.application.SteamMaintenanceTask(id)
}
