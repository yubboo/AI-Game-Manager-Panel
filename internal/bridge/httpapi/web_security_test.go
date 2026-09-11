package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

func bootstrapCookieSession(t *testing.T) (http.Handler, *http.Cookie, *http.Cookie) {
	t.Helper()
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	payload, _ := json.Marshal(authservice.CreateOwnerRequest{Username: "owner", Password: "owner-password"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/bootstrap/owner", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("bootstrap=%d %s", rec.Code, rec.Body.String())
	}
	var sessionCookie, csrfCookie *http.Cookie
	for _, cookie := range rec.Result().Cookies() {
		switch cookie.Name {
		case sessionCookieName:
			sessionCookie = cookie
		case csrfCookieName:
			csrfCookie = cookie
		}
	}
	if sessionCookie == nil || csrfCookie == nil {
		t.Fatalf("missing auth cookies: %v", rec.Header().Values("Set-Cookie"))
	}
	return handler, sessionCookie, csrfCookie
}

func TestWebSessionCookieIsHttpOnlyStrictAndCanAuthenticateReads(t *testing.T) {
	handler, sessionCookie, csrfCookie := bootstrapCookieSession(t)
	if !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie not hardened: %+v", sessionCookie)
	}
	if csrfCookie.HttpOnly || csrfCookie.SameSite != http.SameSiteStrictMode || csrfCookie.Value == "" {
		t.Fatalf("csrf cookie must be readable by same-origin frontend and strict: %+v", csrfCookie)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	req.AddCookie(sessionCookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cookie-authenticated read=%d %s", rec.Code, rec.Body.String())
	}
}

func TestCookieAuthenticatedWritesRequireMatchingCSRF(t *testing.T) {
	handler, sessionCookie, csrfCookie := bootstrapCookieSession(t)

	without := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	without.AddCookie(sessionCookie)
	without.AddCookie(csrfCookie)
	withoutRec := httptest.NewRecorder()
	handler.ServeHTTP(withoutRec, without)
	if withoutRec.Code != http.StatusForbidden {
		t.Fatalf("cookie write without CSRF=%d %s", withoutRec.Code, withoutRec.Body.String())
	}

	wrong := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	wrong.AddCookie(sessionCookie)
	wrong.AddCookie(csrfCookie)
	wrong.Header.Set(csrfHeaderName, "wrong-token")
	wrongRec := httptest.NewRecorder()
	handler.ServeHTTP(wrongRec, wrong)
	if wrongRec.Code != http.StatusForbidden {
		t.Fatalf("cookie write with wrong CSRF=%d %s", wrongRec.Code, wrongRec.Body.String())
	}

	good := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	good.AddCookie(sessionCookie)
	good.AddCookie(csrfCookie)
	good.Header.Set(csrfHeaderName, csrfCookie.Value)
	goodRec := httptest.NewRecorder()
	handler.ServeHTTP(goodRec, good)
	if goodRec.Code != http.StatusNoContent {
		t.Fatalf("cookie write with CSRF=%d %s", goodRec.Code, goodRec.Body.String())
	}
	clearCount := 0
	for _, cookie := range goodRec.Result().Cookies() {
		if (cookie.Name == sessionCookieName || cookie.Name == csrfCookieName) && cookie.MaxAge < 0 {
			clearCount++
		}
	}
	if clearCount != 2 {
		t.Fatalf("logout did not expire both cookies: %v", goodRec.Header().Values("Set-Cookie"))
	}
}

func TestBearerAuthenticatedWritesDoNotNeedBrowserCSRF(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	owner, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{Username: "owner", Password: "owner-password"})
	if err != nil {
		t.Fatal(err)
	}
	rec := request(handler, http.MethodPost, "/api/v1/auth/logout", nil, owner.Token)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("bearer API write unexpectedly required browser CSRF: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSecurityHeadersApplyToPublicAndPrivateResponses(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	for _, path := range []string{"/api/v1/health", "/api/v1/info"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Header().Get("X-Frame-Options") != "DENY" || rec.Header().Get("X-Content-Type-Options") != "nosniff" || rec.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatalf("missing hardening headers for %s: %#v", path, rec.Header())
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatalf("missing CSP frame protection for %s: %q", path, rec.Header().Get("Content-Security-Policy"))
		}
	}
}

func TestForwardedHTTPSOnlyTrustedFromLoopbackProxy(t *testing.T) {
	direct := httptest.NewRequest(http.MethodGet, "http://agmp.test", nil)
	direct.RemoteAddr = "203.0.113.4:44321"
	direct.Header.Set("X-Forwarded-Proto", "https")
	if requestUsesSecureTransport(direct) {
		t.Fatal("direct internet client forged X-Forwarded-Proto=https")
	}
	proxy := httptest.NewRequest(http.MethodGet, "http://127.0.0.1", nil)
	proxy.RemoteAddr = "127.0.0.1:50000"
	proxy.Header.Set("X-Forwarded-Proto", "https")
	if !requestUsesSecureTransport(proxy) {
		t.Fatal("trusted loopback reverse proxy https metadata was ignored")
	}
}

func TestCrossSiteBrowserCannotBootstrapOwner(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	payload := `{"username":"attacker","password":"attacker-password"}`

	cross := httptest.NewRequest(http.MethodPost, "http://agmp.test/api/v1/auth/bootstrap/owner", strings.NewReader(payload))
	cross.Host = "agmp.test"
	cross.Header.Set("Content-Type", "application/json")
	cross.Header.Set("Origin", "https://evil.example")
	cross.Header.Set("Sec-Fetch-Site", "cross-site")
	crossRec := httptest.NewRecorder()
	handler.ServeHTTP(crossRec, cross)
	if crossRec.Code != http.StatusForbidden {
		t.Fatalf("cross-site bootstrap=%d %s", crossRec.Code, crossRec.Body.String())
	}

	same := httptest.NewRequest(http.MethodPost, "http://agmp.test/api/v1/auth/bootstrap/owner", strings.NewReader(payload))
	same.Host = "agmp.test"
	same.Header.Set("Content-Type", "application/json")
	same.Header.Set("Origin", "http://agmp.test")
	same.Header.Set("Sec-Fetch-Site", "same-origin")
	sameRec := httptest.NewRecorder()
	handler.ServeHTTP(sameRec, same)
	if sameRec.Code != http.StatusCreated {
		t.Fatalf("same-origin bootstrap=%d %s", sameRec.Code, sameRec.Body.String())
	}
}

func TestNullOriginIsRejectedForAPIWritesButCLIWithoutOriginWorks(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	payload := `{"username":"owner","password":"owner-password"}`

	nullReq := httptest.NewRequest(http.MethodPost, "http://agmp.test/api/v1/auth/bootstrap/owner", strings.NewReader(payload))
	nullReq.Header.Set("Origin", "null")
	nullRec := httptest.NewRecorder()
	handler.ServeHTTP(nullRec, nullReq)
	if nullRec.Code != http.StatusForbidden {
		t.Fatalf("null-origin bootstrap=%d %s", nullRec.Code, nullRec.Body.String())
	}

	cliReq := httptest.NewRequest(http.MethodPost, "http://agmp.test/api/v1/auth/bootstrap/owner", strings.NewReader(payload))
	cliRec := httptest.NewRecorder()
	handler.ServeHTTP(cliRec, cliReq)
	if cliRec.Code != http.StatusCreated {
		t.Fatalf("non-browser CLI bootstrap=%d %s", cliRec.Code, cliRec.Body.String())
	}
}
