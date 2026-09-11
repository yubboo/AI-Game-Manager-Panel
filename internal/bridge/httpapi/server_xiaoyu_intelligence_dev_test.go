//go:build agmp_dev_license

package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func TestXiaoYuIntelligenceWebRoutesEnforceCoreAndVisibility(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "owner-password-123"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ConfirmMyCredentialStepUp(owner.Token, authservice.CredentialStepUpRequest{Password: "owner-password-123"}); err != nil {
		t.Fatal(err)
	}
	invite, err := app.CreateUserInvitation(owner.Token, authservice.CreateInvitationRequest{Role: authservice.RoleOperator, TargetUsername: "worker"})
	if err != nil {
		t.Fatal(err)
	}
	member, err := app.RegisterInvitedUser(authservice.RegisterInvitationRequest{InvitationToken: invite.Token, Username: "worker", Password: "worker-password-123"})
	if err != nil {
		t.Fatal(err)
	}

	if rec := request(handler, http.MethodGet, "/api/v1/xiaoyu/intelligence", nil, member.Token); rec.Code != http.StatusForbidden {
		t.Fatalf("pending member reached Intelligence core: %d %s", rec.Code, rec.Body.String())
	}
	grant, err := app.CreateMemberCoreAuthorization(owner.Token, authservice.CreateMemberAuthorizationRequest{UserID: member.User.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.RedeemMyCoreAuthorization(member.Token, authservice.RedeemMemberAuthorizationRequest{Token: grant.Token}); err != nil {
		t.Fatal(err)
	}

	catalog := request(handler, http.MethodGet, "/api/v1/xiaoyu/intelligence", nil, member.Token)
	if catalog.Code != http.StatusOK || !strings.Contains(catalog.Body.String(), "饥荒联机版专家") || !strings.Contains(catalog.Body.String(), "Minecraft 专家") {
		t.Fatalf("authorized member missing builtin intelligence: %d %s", catalog.Code, catalog.Body.String())
	}
	privatePayload, _ := json.Marshal(xiaoyuhost.SkillSaveRequest{Name: "worker private", Prompt: "only worker", Visibility: xiaoyuhost.VisibilityPrivate})
	privateRec := request(handler, http.MethodPost, "/api/v1/xiaoyu/intelligence/skills", privatePayload, member.Token)
	if privateRec.Code != http.StatusOK {
		t.Fatalf("private Skill save=%d %s", privateRec.Code, privateRec.Body.String())
	}

	sharedPayload, _ := json.Marshal(xiaoyuhost.SkillSaveRequest{Name: "worker shared", Prompt: "must fail", Visibility: xiaoyuhost.VisibilityOrganization})
	sharedRec := request(handler, http.MethodPost, "/api/v1/xiaoyu/intelligence/skills", sharedPayload, member.Token)
	if sharedRec.Code != http.StatusForbidden {
		t.Fatalf("operator published organization Skill: %d %s", sharedRec.Code, sharedRec.Body.String())
	}

	ownerCatalog := request(handler, http.MethodGet, "/api/v1/xiaoyu/intelligence", nil, owner.Token)
	if ownerCatalog.Code != http.StatusOK {
		t.Fatalf("owner catalog=%d %s", ownerCatalog.Code, ownerCatalog.Body.String())
	}
	if strings.Contains(ownerCatalog.Body.String(), "worker private") {
		t.Fatal("Owner supervision leaked member private Intelligence")
	}
}
