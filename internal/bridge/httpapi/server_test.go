package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

func TestRequireLoopback(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:17890", "localhost:17890", "[::1]:17890"} {
		if err := requireLoopback(addr); err != nil {
			t.Fatalf("%s should be allowed: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:17890", "192.168.1.10:17890"} {
		if err := requireLoopback(addr); err == nil {
			t.Fatalf("%s should stay rejected until remote transport security is explicitly enabled", addr)
		}
	}
}

func TestRemoteListenRequiresExplicitOptIn(t *testing.T) {
	if err := validateListenPolicy("0.0.0.0:17890", false); err == nil {
		t.Fatal("non-loopback must stay blocked unless remote Web is explicitly enabled")
	}
	if err := validateListenPolicy("0.0.0.0:17890", true); err != nil {
		t.Fatalf("explicit remote Web opt-in should allow non-loopback bind: %v", err)
	}
	if err := validateListenPolicy("127.0.0.1:17890", false); err != nil {
		t.Fatalf("loopback should remain allowed by default: %v", err)
	}
}

func TestBearerSessionGateProtectsBusinessAPI(t *testing.T) {
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{
		Root:    root,
		DataDir: "data",
	})
	handler := New(app, Options{}).routes()

	assertStatus(t, handler, http.MethodGet, "/api/v1/health", nil, "", http.StatusOK)
	assertStatus(t, handler, http.MethodGet, "/api/v1/license/status", nil, "", http.StatusUnauthorized)
	assertStatus(t, handler, http.MethodPost, "/api/v1/license/activate", []byte(`{"cdk":"BFCDK2.invalid"}`), "", http.StatusUnauthorized)
	assertStatus(t, handler, http.MethodGet, "/api/v1/auth/bootstrap", nil, "", http.StatusOK)
	assertStatus(t, handler, http.MethodGet, "/api/v1/info", nil, "", http.StatusUnauthorized)

	ownerPayload, _ := json.Marshal(authservice.CreateOwnerRequest{
		Username:    "owner",
		DisplayName: "Owner",
		Password:    "agmp-test-password",
	})
	recorder := request(handler, http.MethodPost, "/api/v1/auth/bootstrap/owner", ownerPayload, "")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("bootstrap owner status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var session authservice.Session
	if err := json.Unmarshal(recorder.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode bootstrap session: %v", err)
	}
	if session.Token == "" {
		t.Fatal("bootstrap session token should not be empty")
	}

	assertStatus(t, handler, http.MethodGet, "/api/v1/info", nil, session.Token, http.StatusOK)
	assertStatus(t, handler, http.MethodGet, "/api/v1/license/status", nil, session.Token, http.StatusOK)
	displayNamePayload, _ := json.Marshal(authservice.UpdateDisplayNameRequest{DisplayName: "AI游戏管理器面板管理员"})
	assertStatus(t, handler, http.MethodPut, "/api/v1/auth/me/display-name", displayNamePayload, session.Token, http.StatusOK)
	me := request(handler, http.MethodGet, "/api/v1/auth/me", nil, session.Token)
	var updatedUser authservice.User
	if err := json.Unmarshal(me.Body.Bytes(), &updatedUser); err != nil || updatedUser.DisplayName != "AI游戏管理器面板管理员" {
		t.Fatalf("display name update not reflected in session: user=%+v err=%v", updatedUser, err)
	}
	assertStatus(t, handler, http.MethodPost, "/api/v1/auth/bootstrap/owner", ownerPayload, "", http.StatusConflict)
	assertStatus(t, handler, http.MethodPost, "/api/v1/auth/logout", nil, session.Token, http.StatusNoContent)
	assertStatus(t, handler, http.MethodGet, "/api/v1/info", nil, session.Token, http.StatusUnauthorized)
}

func assertStatus(t *testing.T, handler http.Handler, method, path string, body []byte, token string, want int) {
	t.Helper()
	recorder := request(handler, method, path, body, token)
	if recorder.Code != want {
		t.Fatalf("%s %s status = %d, want %d, body=%s", method, path, recorder.Code, want, recorder.Body.String())
	}
}

func request(handler http.Handler, method, path string, body []byte, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func TestSecurityKeyAndEnvironmentSkipRoutes(t *testing.T) {
	root := t.TempDir()
	app := application.NewWithOptions(application.Options{Root: root, DataDir: "data"})
	handler := New(app, Options{}).routes()

	generated := request(handler, http.MethodPost, "/api/v1/auth/security-key/generate", nil, "")
	if generated.Code != http.StatusOK {
		t.Fatalf("generate key status=%d body=%s", generated.Code, generated.Body.String())
	}
	var material authservice.SecurityKeyMaterial
	if err := json.Unmarshal(generated.Body.Bytes(), &material); err != nil || material.Key == "" {
		t.Fatalf("invalid generated key response: %+v err=%v", material, err)
	}

	ownerPayload, _ := json.Marshal(authservice.CreateOwnerRequest{
		Username: "owner", Password: "agmp-test-password", SecurityKey: material.Key, RequireSecurityKey: true,
	})
	created := request(handler, http.MethodPost, "/api/v1/auth/bootstrap/owner", ownerPayload, "")
	if created.Code != http.StatusCreated {
		t.Fatalf("create owner status=%d body=%s", created.Code, created.Body.String())
	}
	var session authservice.Session
	if err := json.Unmarshal(created.Body.Bytes(), &session); err != nil {
		t.Fatal(err)
	}

	securityRecorder := request(handler, http.MethodGet, "/api/v1/auth/security-key/status", nil, session.Token)
	if securityRecorder.Code != http.StatusOK {
		t.Fatalf("security status=%d body=%s", securityRecorder.Code, securityRecorder.Body.String())
	}
	var securityStatus authservice.SecurityStatus
	if err := json.Unmarshal(securityRecorder.Body.Bytes(), &securityStatus); err != nil || !securityStatus.Configured || !securityStatus.VerificationEnabled {
		t.Fatalf("unexpected security status: %+v err=%v", securityStatus, err)
	}

	skipRecorder := request(handler, http.MethodPost, "/api/v1/environment/setup/skip", nil, session.Token)
	if skipRecorder.Code != http.StatusOK {
		t.Fatalf("skip environment status=%d body=%s", skipRecorder.Code, skipRecorder.Body.String())
	}
	var skipped struct {
		Skipped bool `json:"skipped"`
	}
	if err := json.Unmarshal(skipRecorder.Body.Bytes(), &skipped); err != nil || !skipped.Skipped {
		t.Fatalf("environment skip was not persisted in response: %+v err=%v", skipped, err)
	}
	assertStatus(t, handler, http.MethodPost, "/api/v1/auth/logout", nil, session.Token, http.StatusNoContent)

	loginWithoutKey, _ := json.Marshal(authservice.LoginRequest{Username: "owner", Password: "agmp-test-password"})
	assertStatus(t, handler, http.MethodPost, "/api/v1/auth/login", loginWithoutKey, "", http.StatusUnauthorized)
	loginWithKey, _ := json.Marshal(authservice.LoginRequest{Username: "owner", Password: "agmp-test-password", SecurityKey: material.Key})
	assertStatus(t, handler, http.MethodPost, "/api/v1/auth/login", loginWithKey, "", http.StatusOK)
}
