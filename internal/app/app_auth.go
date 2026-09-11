package app

import (
	"fmt"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	licenseservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
)

func (a *Application) AuthBootstrapStatus() authservice.BootstrapStatus {
	return a.auth.BootstrapStatus()
}
func (a *Application) GenerateSecurityKey() (authservice.SecurityKeyMaterial, error) {
	return authservice.GenerateSecurityKey()
}
func (a *Application) CreateInitialAdministrator(request authservice.CreateOwnerRequest) (authservice.Session, error) {
	session, err := a.auth.CreateInitialOwner(request)
	if err != nil {
		a.logger.Warn("initial administrator creation failed", "error", err.Error())
		return session, err
	}
	a.recordOperation("info", "auth", "bootstrap_owner_created", session.User.Username, "success", "首个最高管理员已创建；公开注册永久关闭", "", "")
	return session, nil
}
func (a *Application) Login(request authservice.LoginRequest) (authservice.Session, error) {
	session, err := a.auth.Login(request)
	if err != nil {
		a.logger.Warn("login failed", "username", request.Username)
		return session, err
	}
	a.recordOperation("info", "auth", "login", session.User.Username, "success", "组织成员已登录", "", "")
	return session, nil
}
func (a *Application) ValidateSession(token string) (authservice.User, error) {
	return a.auth.Validate(token)
}
func (a *Application) Logout(token string) {
	if user, err := a.auth.Validate(token); err == nil {
		a.recordOperation("info", "auth", "logout", user.Username, "success", "用户已退出", "", "")
	}
	a.auth.Logout(token)
}
func (a *Application) ListUsers(token string) ([]authservice.User, error) {
	return a.auth.ListUsers(token)
}
func (a *Application) UpdateMyDisplayName(token string, request authservice.UpdateDisplayNameRequest) (authservice.User, error) {
	user, err := a.auth.UpdateMyDisplayName(token, request)
	if err == nil {
		a.recordOperation("info", "auth", "display_name_updated", user.Username, "success", "用户修改了显示名称", "", "")
	}
	return user, err
}
func (a *Application) CreateUser(token string, request authservice.CreateUserRequest) (authservice.User, error) {
	return a.auth.CreateUser(token, request)
}
func (a *Application) MySecurityStatus(token string) (authservice.SecurityStatus, error) {
	return a.auth.MySecurityStatus(token)
}
func (a *Application) RotateMySecurityKey(token string, request authservice.RotateSecurityKeyRequest) (authservice.SecurityKeyMaterial, error) {
	material, err := a.auth.RotateMySecurityKey(token, request)
	if err != nil {
		return material, err
	}
	if user, e := a.auth.Validate(token); e == nil {
		a.recordOperation("info", "auth", "security_key_rotated", user.Username, "success", "登录安全密钥已重新生成", "", "")
	}
	return material, nil
}
func (a *Application) SetMySecurityKeyVerification(token string, request authservice.SetSecurityKeyVerificationRequest) (authservice.SecurityStatus, error) {
	status, err := a.auth.SetMySecurityKeyVerification(token, request)
	if err != nil {
		return status, err
	}
	if user, e := a.auth.Validate(token); e == nil {
		message := "登录安全密钥验证已关闭"
		if status.VerificationEnabled {
			message = "登录安全密钥验证已开启"
		}
		a.recordOperation("info", "auth", "security_key_verification", user.Username, "success", message, "", "")
	}
	return status, nil
}

func (a *Application) AuthInstanceIdentity() authservice.InstanceIdentity {
	return a.auth.InstanceIdentity()
}
func (a *Application) AuthOrganization(token string) (authservice.Organization, error) {
	return a.auth.Organization(token)
}
func (a *Application) CreateUserInvitation(token string, request authservice.CreateInvitationRequest) (authservice.CreatedInvitation, error) {
	v, err := a.auth.CreateInvitation(token, request)
	if err != nil {
		return v, err
	}
	a.recordOperation("warn", "auth", "member_invitation_created", v.ID, "success", "管理员签发了组织成员邀请", "", "")
	return v, nil
}
func (a *Application) ListUserInvitations(token string) ([]authservice.InvitationView, error) {
	return a.auth.ListInvitations(token)
}
func (a *Application) RevokeUserInvitation(token, id string) (authservice.InvitationView, error) {
	v, err := a.auth.RevokeInvitation(token, id)
	if err == nil {
		a.recordOperation("warn", "auth", "member_invitation_revoked", v.ID, "success", "管理员撤销了组织成员邀请", "", "")
	}
	return v, err
}
func (a *Application) InspectUserInvitation(token string) (authservice.InvitationPreview, error) {
	return a.auth.InspectInvitation(token)
}
func (a *Application) RegisterInvitedUser(request authservice.RegisterInvitationRequest) (authservice.Session, error) {
	session, err := a.auth.RegisterWithInvitation(request)
	if err != nil {
		return session, err
	}
	a.recordOperation("info", "auth", "invited_member_registered", session.User.Username, "success", "受邀成员完成注册并加入组织", "", "")
	return session, nil
}

func (a *Application) MyEmailSecurityStatus(token string) (authservice.EmailSecurityStatus, error) {
	return a.auth.MyEmailSecurityStatus(token)
}
func (a *Application) BindMyEmail(token string, request authservice.BindEmailRequest) (authservice.EmailSecurityStatus, error) {
	status, err := a.auth.BindMyEmail(token, request)
	if err == nil {
		a.recordOperation("info", "auth", "email_bound", "self", "success", "账号绑定了可选邮箱；等待用户验证", "", "")
	}
	return status, err
}
func (a *Application) UnbindMyEmail(token, password string) (authservice.EmailSecurityStatus, error) {
	status, err := a.auth.UnbindMyEmail(token, password)
	if err == nil {
		a.recordOperation("warn", "auth", "email_unbound", "self", "success", "账号取消了邮箱绑定", "", "")
	}
	return status, err
}
func (a *Application) RequestMyEmailVerification(token string, request authservice.RequestEmailVerificationRequest) error {
	return a.auth.RequestEmailVerification(token, request)
}
func (a *Application) ConfirmMyEmailVerification(token string, request authservice.ConfirmEmailVerificationRequest) (authservice.EmailSecurityStatus, error) {
	status, err := a.auth.ConfirmEmailVerification(token, request)
	if err == nil {
		a.recordOperation("info", "auth", "email_verification_completed", "self", "success", "邮箱验证/风险二次验证已完成", "", "")
	}
	return status, err
}
func (a *Application) ConfirmMyCredentialStepUp(token string, request authservice.CredentialStepUpRequest) (authservice.User, error) {
	user, err := a.auth.ConfirmCredentialStepUp(token, request)
	if err == nil {
		a.recordOperation("info", "auth", "credential_step_up", user.Username, "success", "高风险操作凭据二次验证已完成", "", "")
	}
	return user, err
}
func (a *Application) RequestPasswordReset(request authservice.RequestPasswordResetRequest) (authservice.PasswordResetRequestStatus, error) {
	return a.auth.RequestPasswordReset(request)
}
func (a *Application) ConfirmPasswordReset(request authservice.ConfirmPasswordResetRequest) error {
	return a.auth.ConfirmPasswordReset(request)
}

func (a *Application) SMTPSettings(token string) (authservice.SMTPSettings, error) {
	return a.auth.SMTPSettings(token)
}
func (a *Application) SaveSMTPSettings(token string, request authservice.SaveSMTPSettingsRequest) (authservice.SMTPSettings, error) {
	status, err := a.auth.SaveSMTPSettings(token, request)
	if err == nil {
		a.recordOperation("warn", "auth", "smtp_settings_updated", "email-security", "success", "管理员更新了系统邮箱服务配置", "", "")
	}
	return status, err
}

func (a *Application) CreateMemberCoreAuthorization(token string, request authservice.CreateMemberAuthorizationRequest) (authservice.CreatedMemberAuthorization, error) {
	status := a.license.Status()
	if status.State != licenseservice.StateDevelopment {
		if err := a.license.RequireFeature("server.basic"); err != nil {
			return authservice.CreatedMemberAuthorization{}, err
		}
		if status.SeatLimit > 0 {
			users, err := a.auth.ListUsers(token)
			if err != nil {
				return authservice.CreatedMemberAuthorization{}, err
			}
			already := false
			for _, u := range users {
				if u.ID == request.UserID && (u.Role == authservice.RoleOwner || u.CoreAccess == authservice.CoreAccessAuthorized) {
					already = true
					break
				}
			}
			if !already && a.auth.CoreSeatCount() >= status.SeatLimit {
				return authservice.CreatedMemberAuthorization{}, fmt.Errorf("许可证成员席位已用满：%d/%d，请回收成员核心授权或升级许可证", a.auth.CoreSeatCount(), status.SeatLimit)
			}
		}
	}
	value, err := a.auth.CreateMemberAuthorization(token, request)
	if err == nil {
		a.recordOperation("warn", "auth", "member_core_authorization_created", value.UserID, "success", "超级管理员签发了成员核心功能授权码", "", "")
	}
	return value, err
}
func (a *Application) ListMemberCoreAuthorizations(token string) ([]authservice.MemberAuthorizationView, error) {
	return a.auth.ListMemberAuthorizations(token)
}
func (a *Application) RevokeMemberCoreAuthorization(token, id string) (authservice.MemberAuthorizationView, error) {
	v, err := a.auth.RevokeMemberAuthorization(token, id)
	if err == nil {
		a.recordOperation("warn", "auth", "member_core_authorization_code_revoked", v.UserID, "success", "超级管理员撤销了未使用的成员核心授权码", "", "")
	}
	return v, err
}
func (a *Application) RedeemMyCoreAuthorization(token string, request authservice.RedeemMemberAuthorizationRequest) (authservice.User, error) {
	user, err := a.auth.RedeemMemberAuthorization(token, request)
	if err == nil {
		a.recordOperation("warn", "auth", "member_core_authorization_redeemed", user.Username, "success", "成员完成核心功能授权", "", "")
	}
	return user, err
}
func (a *Application) RevokeMemberCoreAccess(token, userID string) (authservice.User, error) {
	user, err := a.auth.RevokeMemberCoreAccess(token, userID)
	if err == nil {
		a.recordOperation("warn", "auth", "member_core_access_revoked", user.Username, "success", "超级管理员暂停了成员核心功能授权", "", "")
	}
	return user, err
}
func (a *Application) ClearMemberRisk(token, userID string) (authservice.User, error) {
	user, err := a.auth.ClearUserRisk(token, userID)
	if err == nil {
		a.recordOperation("warn", "auth", "member_risk_cleared", user.Username, "success", "超级管理员解除成员风控；后续高风险操作仍需成员本人二次验证", "", "")
	}
	return user, err
}
func (a *Application) RemoveOrganizationMember(token, userID string) error {
	err := a.auth.RemoveMember(token, userID)
	if err == nil {
		a.recordOperation("warn", "auth", "member_removed", userID, "success", "成员已移出组织，会话立即失效", "", "")
	}
	return err
}
func (a *Application) RequireMemberCoreAccess(token string) (authservice.User, error) {
	return a.auth.RequireCoreAccess(token)
}

// RequireOrganizationAdministrator centralizes organization-wide configuration
// authorization for every bridge. Keeping this in Application prevents Web and
// desktop adapters from drifting into different role rules.
func (a *Application) RequireOrganizationAdministrator(token string) (authservice.User, error) {
	user, err := a.auth.Validate(token)
	if err != nil {
		return authservice.User{}, err
	}
	if user.Role != authservice.RoleOwner && user.Role != authservice.RoleAdministrator {
		return authservice.User{}, authservice.ErrForbidden
	}
	return user, nil
}

func (a *Application) RequireSensitiveAdministrator(token, reason string) (authservice.User, error) {
	user, err := a.auth.RequireSensitiveAction(token, reason)
	if err != nil {
		return authservice.User{}, err
	}
	if user.Role != authservice.RoleOwner && user.Role != authservice.RoleAdministrator {
		return authservice.User{}, authservice.ErrForbidden
	}
	return user, nil
}

func (a *Application) RequireSensitiveOwner(token, reason string) (authservice.User, error) {
	user, err := a.auth.RequireSensitiveAction(token, reason)
	if err != nil {
		return authservice.User{}, err
	}
	if user.Role != authservice.RoleOwner {
		return authservice.User{}, authservice.ErrForbidden
	}
	return user, nil
}
