package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

type httpAccessFixture struct {
	handler  http.Handler
	owner    authservice.Session
	admin    authservice.Session
	operator authservice.Session
}

func newHTTPAccessFixture(t *testing.T) httpAccessFixture {
	t.Helper()
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	ownerPayload, _ := json.Marshal(authservice.CreateOwnerRequest{Username: "owner", Password: "owner-password"})
	ownerRec := request(handler, http.MethodPost, "/api/v1/auth/bootstrap/owner", ownerPayload, "")
	if ownerRec.Code != http.StatusCreated {
		t.Fatalf("owner bootstrap=%d %s", ownerRec.Code, ownerRec.Body.String())
	}
	var owner authservice.Session
	if err := json.Unmarshal(ownerRec.Body.Bytes(), &owner); err != nil {
		t.Fatal(err)
	}
	stepPayload, _ := json.Marshal(authservice.CredentialStepUpRequest{Password: "owner-password"})
	if rec := request(handler, http.MethodPost, "/api/v1/auth/step-up/credentials", stepPayload, owner.Token); rec.Code != http.StatusOK {
		t.Fatalf("owner step-up=%d %s", rec.Code, rec.Body.String())
	}
	register := func(role, username, password string) authservice.Session {
		t.Helper()
		invitePayload, _ := json.Marshal(authservice.CreateInvitationRequest{Role: role, TargetUsername: username})
		inviteRec := request(handler, http.MethodPost, "/api/v1/users/invitations", invitePayload, owner.Token)
		if inviteRec.Code != http.StatusCreated {
			t.Fatalf("invite %s=%d %s", username, inviteRec.Code, inviteRec.Body.String())
		}
		var invite authservice.CreatedInvitation
		if err := json.Unmarshal(inviteRec.Body.Bytes(), &invite); err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(authservice.RegisterInvitationRequest{InvitationToken: invite.Token, Username: username, Password: password})
		rec := request(handler, http.MethodPost, "/api/v1/auth/invitation/register", payload, "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("register %s=%d %s", username, rec.Code, rec.Body.String())
		}
		var session authservice.Session
		if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
			t.Fatal(err)
		}
		return session
	}
	return httpAccessFixture{
		handler:  handler,
		owner:    owner,
		admin:    register(authservice.RoleAdministrator, "admin", "admin-password"),
		operator: register(authservice.RoleOperator, "operator", "operator-password"),
	}
}

func TestSystemConfigurationRequiresOrganizationAdministrator(t *testing.T) {
	fx := newHTTPAccessFixture(t)
	settingsPayload := []byte(`{"theme":"dark","language":"zh-CN","debug":false}`)
	assertStatus(t, fx.handler, http.MethodGet, "/api/v1/settings", nil, fx.operator.Token, http.StatusOK)
	assertStatus(t, fx.handler, http.MethodPut, "/api/v1/settings", settingsPayload, fx.operator.Token, http.StatusForbidden)
	assertStatus(t, fx.handler, http.MethodPut, "/api/v1/settings", settingsPayload, fx.admin.Token, http.StatusNoContent)

	assertStatus(t, fx.handler, http.MethodPost, "/api/v1/environment/setup/skip", nil, fx.operator.Token, http.StatusForbidden)
	assertStatus(t, fx.handler, http.MethodPost, "/api/v1/environment/setup/skip", nil, fx.admin.Token, http.StatusOK)

	assertStatus(t, fx.handler, http.MethodGet, "/api/v1/environment/runtime/catalog", nil, fx.operator.Token, http.StatusOK)
	javaPayload := []byte(`{"major":21}`)
	assertStatus(t, fx.handler, http.MethodPost, "/api/v1/environment/runtime/java/install", javaPayload, fx.operator.Token, http.StatusForbidden)
	registerPayload := []byte(`{"kind":"steamcmd","executable":"/definitely/missing/steamcmd"}`)
	adminRegister := request(fx.handler, http.MethodPost, "/api/v1/environment/runtime/register", registerPayload, fx.admin.Token)
	if adminRegister.Code == http.StatusForbidden || adminRegister.Code == http.StatusUnauthorized {
		t.Fatalf("administrator should pass Runtime Manager RBAC before path validation: %d %s", adminRegister.Code, adminRegister.Body.String())
	}

	smtpPayload := []byte(`{"host":"127.0.0.1","port":2525,"from":"agmp@example.test","tlsMode":"none","accountPassword":"operator-password"}`)
	assertStatus(t, fx.handler, http.MethodPut, "/api/v1/settings/email", smtpPayload, fx.operator.Token, http.StatusForbidden)
	smtpAdminPayload := []byte(`{"host":"127.0.0.1","port":2525,"from":"agmp@example.test","tlsMode":"none","accountPassword":"admin-password"}`)
	assertStatus(t, fx.handler, http.MethodPut, "/api/v1/settings/email", smtpAdminPayload, fx.admin.Token, http.StatusOK)
}

func TestLicenseMutationIsOwnerOnlyAndLogDeletionIsAdministrative(t *testing.T) {
	fx := newHTTPAccessFixture(t)
	invalidLicense := []byte(`{"cdk":"BFCDK2.invalid"}`)
	assertStatus(t, fx.handler, http.MethodPost, "/api/v1/license/activate", invalidLicense, fx.operator.Token, http.StatusForbidden)
	// Owner reaches the license verifier rather than being blocked by RBAC.
	assertStatus(t, fx.handler, http.MethodPost, "/api/v1/license/activate", invalidLicense, fx.owner.Token, http.StatusBadRequest)

	assertStatus(t, fx.handler, http.MethodDelete, "/api/v1/loghub/logs/nonexistent", nil, fx.operator.Token, http.StatusForbidden)
	adminDelete := request(fx.handler, http.MethodDelete, "/api/v1/loghub/logs/nonexistent", nil, fx.admin.Token)
	if adminDelete.Code == http.StatusForbidden || adminDelete.Code == http.StatusUnauthorized {
		t.Fatalf("administrator should pass RBAC before log lookup: %d %s", adminDelete.Code, adminDelete.Body.String())
	}
}
