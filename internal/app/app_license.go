package app

import (
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	licenseservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/license"
)

func (a *Application) LicenseStatus() licenseservice.Status {
	status := a.license.Status()
	if a.auth != nil {
		status.ActiveSeats = a.auth.CoreSeatCount()
	}
	return status
}
func (a *Application) ActivateLicense(request licenseservice.ActivateRequest) (licenseservice.Status, error) {
	status, err := a.license.Activate(request)
	if err != nil {
		a.logger.Warn("license activation failed", "error", err.Error())
		return status, err
	}
	a.recordOperation("info", "license", "activate", status.LicenseID, "success", "许可证已绑定当前设备", "", "")
	return status, nil
}
func (a *Application) UnbindLicense() (licenseservice.Status, error) {
	status, err := a.license.Unbind()
	if err != nil {
		return status, err
	}
	a.recordOperation("info", "license", "unbind", "current-device", "success", "已解绑当前设备本地许可证", "", "")
	return status, nil
}
func (a *Application) LicenseFeatureStatus(feature string) licenseservice.FeatureStatus {
	return a.license.HasFeature(feature)
}
func (a *Application) RequireLicenseFeature(feature string) error {
	return a.license.RequireFeature(feature)
}
func (a *Application) RequireLicensedMemberFeature(token, feature string) (authservice.User, error) {
	if err := a.license.RequireFeature(feature); err != nil {
		return authservice.User{}, err
	}
	return a.auth.RequireCoreAccess(token)
}

func (a *Application) ActivateLicenseForUser(token string, request licenseservice.ActivateRequest) (licenseservice.Status, error) {
	// Initial product activation must remain possible before optional email or a
	// separate step-up provider exists. Owner role + a valid signed activation
	// artifact is the gate; unbinding/replacing an active trust relationship is
	// handled as a separate sensitive action below.
	user, err := a.ValidateSession(token)
	if err != nil {
		return a.LicenseStatus(), err
	}
	if user.Role != authservice.RoleOwner {
		return a.LicenseStatus(), authservice.ErrForbidden
	}
	return a.ActivateLicense(request)
}

func (a *Application) UnbindLicenseForUser(token string) (licenseservice.Status, error) {
	if _, err := a.RequireSensitiveOwner(token, "解绑 AGMP 官方许可证"); err != nil {
		return a.LicenseStatus(), err
	}
	return a.UnbindLicense()
}
