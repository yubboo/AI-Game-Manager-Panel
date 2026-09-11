package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	application "github.com/yubboo/AI-Game-Manager-Panel/internal/app"
)

func TestAuthClientIPIgnoresSpoofedForwardingHeadersFromDirectClient(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://agmp.test/api/v1/auth/login", nil)
	req.RemoteAddr = "203.0.113.44:52123"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	req.Header.Set("X-Real-IP", "198.51.100.10")
	if got := authClientIP(req); got != "203.0.113.44" {
		t.Fatalf("direct client spoofed auth limiter identity: got %q", got)
	}
}

func TestAuthClientIPAcceptsLocalReverseProxyClientAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/auth/login", nil)
	req.RemoteAddr = "127.0.0.1:49152"
	req.Header.Set("X-Real-IP", "203.0.113.88")
	if got := authClientIP(req); got != "203.0.113.88" {
		t.Fatalf("loopback proxy client IP = %q", got)
	}
}

func TestPublicLoginRateLimitRejectsThirteenthAttempt(t *testing.T) {
	app := application.NewWithOptions(application.Options{Root: t.TempDir(), DataDir: "data"})
	handler := New(app, Options{}).routes()
	body := []byte(`{"username":"nobody","password":"definitely-wrong","securityKey":""}`)
	for i := 0; i < 12; i++ {
		rec := request(handler, http.MethodPost, "/api/v1/auth/login", body, "")
		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("rate limit triggered early on attempt %d", i+1)
		}
	}
	rec := request(handler, http.MethodPost, "/api/v1/auth/login", body, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("13th login attempt status=%d want=429 body=%s", rec.Code, rec.Body.String())
	}
}
