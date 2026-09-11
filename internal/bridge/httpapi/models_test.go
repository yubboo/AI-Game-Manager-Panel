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

func TestXiaoYuModelCenterRoutesPersistProfilesWithoutReturningSecrets(t *testing.T) {
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{Root: root, DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "agmp-test-password"})
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"name":"Local Brain","provider":"custom","protocol":"openai-compatible","baseUrl":"http://127.0.0.1:11434/v1","model":"local-model","apiKey":"should-never-return","enabled":true,"contextWindow":32768,"maxOutputTokens":4096,"thinkingMode":"default"}`)
	savedRecorder := request(handler, http.MethodPost, "/api/v1/xiaoyu/models", payload, owner.Token)
	if savedRecorder.Code != http.StatusOK {
		t.Fatalf("save model status=%d body=%s", savedRecorder.Code, savedRecorder.Body.String())
	}
	if strings.Contains(savedRecorder.Body.String(), "should-never-return") {
		t.Fatal("model save response leaked API key")
	}
	var saved xiaoyuhost.ModelProfileView
	if err := json.Unmarshal(savedRecorder.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" || !saved.HasAPIKey || !saved.IsDefault {
		t.Fatalf("unexpected saved model: %+v", saved)
	}

	catalogRecorder := request(handler, http.MethodGet, "/api/v1/xiaoyu/models", nil, owner.Token)
	if catalogRecorder.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogRecorder.Code, catalogRecorder.Body.String())
	}
	if strings.Contains(catalogRecorder.Body.String(), "should-never-return") {
		t.Fatal("model catalog leaked API key")
	}
	var catalog xiaoyuhost.ModelCatalog
	if err := json.Unmarshal(catalogRecorder.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Profiles) != 1 || catalog.DefaultBrainID != saved.ID {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
}

func TestXiaoYuModelCenterRequiresAdministratorForMutation(t *testing.T) {
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{Root: root, DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "agmp-test-password"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.ConfirmMyCredentialStepUp(owner.Token, authservice.CredentialStepUpRequest{Password: "agmp-test-password"}); err != nil {
		t.Fatal(err)
	}
	invite, err := app.CreateUserInvitation(owner.Token, authservice.CreateInvitationRequest{Role: authservice.RoleOperator, TargetUsername: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	operatorSession, err := app.RegisterInvitedUser(authservice.RegisterInvitationRequest{InvitationToken: invite.Token, Username: "operator", DisplayName: "Operator", Password: "agmp-operator-password"})
	if err != nil {
		t.Fatal(err)
	}
	grant, err := app.CreateMemberCoreAuthorization(owner.Token, authservice.CreateMemberAuthorizationRequest{UserID: operatorSession.User.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.RedeemMyCoreAuthorization(operatorSession.Token, authservice.RedeemMemberAuthorizationRequest{Token: grant.Token}); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"name":"Local","provider":"ollama","protocol":"openai-compatible","baseUrl":"http://127.0.0.1:11434/v1","model":"qwen","enabled":true}`)
	recorder := request(handler, http.MethodPost, "/api/v1/xiaoyu/models", payload, operatorSession.Token)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("operator must not mutate model center: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	assertStatus(t, handler, http.MethodGet, "/api/v1/xiaoyu/models", nil, operatorSession.Token, http.StatusOK)
}
