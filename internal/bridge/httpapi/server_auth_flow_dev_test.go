//go:build agmp_dev_license

package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

func TestInviteMemberCoreAuthorizationFlow(t *testing.T) {
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
	invitePayload, _ := json.Marshal(authservice.CreateInvitationRequest{Role: authservice.RoleOperator, TargetUsername: "worker", ExpiresInHours: 24})
	inviteRec := request(handler, http.MethodPost, "/api/v1/users/invitations", invitePayload, owner.Token)
	if inviteRec.Code != http.StatusCreated {
		t.Fatalf("invite=%d %s", inviteRec.Code, inviteRec.Body.String())
	}
	var invite authservice.CreatedInvitation
	if err := json.Unmarshal(inviteRec.Body.Bytes(), &invite); err != nil {
		t.Fatal(err)
	}

	inspectPayload, _ := json.Marshal(map[string]string{"token": invite.Token})
	if rec := request(handler, http.MethodPost, "/api/v1/auth/invitation/inspect", inspectPayload, ""); rec.Code != http.StatusOK {
		t.Fatalf("inspect=%d %s", rec.Code, rec.Body.String())
	}
	registerPayload, _ := json.Marshal(authservice.RegisterInvitationRequest{InvitationToken: invite.Token, Username: "worker", Password: "worker-password"})
	registerRec := request(handler, http.MethodPost, "/api/v1/auth/invitation/register", registerPayload, "")
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register=%d %s", registerRec.Code, registerRec.Body.String())
	}
	var member authservice.Session
	if err := json.Unmarshal(registerRec.Body.Bytes(), &member); err != nil {
		t.Fatal(err)
	}
	if member.User.CoreAccess != authservice.CoreAccessPending {
		t.Fatalf("new member core=%q", member.User.CoreAccess)
	}

	if rec := request(handler, http.MethodGet, "/api/v1/xiaoyu/models", nil, member.Token); rec.Code != http.StatusForbidden {
		t.Fatalf("pending member reached XiaoYu core: %d %s", rec.Code, rec.Body.String())
	}
	grantPayload, _ := json.Marshal(authservice.CreateMemberAuthorizationRequest{UserID: member.User.ID, ExpiresInHours: 24})
	grantRec := request(handler, http.MethodPost, "/api/v1/users/core-authorizations", grantPayload, owner.Token)
	if grantRec.Code != http.StatusCreated {
		t.Fatalf("grant=%d %s", grantRec.Code, grantRec.Body.String())
	}
	var grant authservice.CreatedMemberAuthorization
	if err := json.Unmarshal(grantRec.Body.Bytes(), &grant); err != nil {
		t.Fatal(err)
	}
	redeemPayload, _ := json.Marshal(authservice.RedeemMemberAuthorizationRequest{Token: grant.Token})
	if rec := request(handler, http.MethodPost, "/api/v1/auth/core/redeem", redeemPayload, member.Token); rec.Code != http.StatusOK {
		t.Fatalf("redeem=%d %s", rec.Code, rec.Body.String())
	}
	if rec := request(handler, http.MethodGet, "/api/v1/xiaoyu/models", nil, member.Token); rec.Code != http.StatusOK {
		t.Fatalf("authorized member XiaoYu core=%d %s", rec.Code, rec.Body.String())
	}
}
