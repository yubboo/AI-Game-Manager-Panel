package httpapi

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
)

const (
	sessionCookieName = "agmp_session"
	csrfCookieName    = "agmp_csrf"
	csrfHeaderName    = "X-AGMP-CSRF"
)

func sessionTokenFromRequest(r *http.Request) (string, bool) {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) >= 8 && strings.EqualFold(value[:7], "Bearer ") {
		if token := strings.TrimSpace(value[7:]); token != "" {
			return token, false
		}
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value), true
	}
	return "", false
}

func requestUsesSecureTransport(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// Only trust proxy protocol metadata when the TCP peer is loopback. This is
	// the same trust boundary used by authClientIP: public clients cannot forge
	// X-Forwarded-Proto to alter cookie security semantics.
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil && ip.IsLoopback() {
		return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https")
	}
	return false
}

func randomCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Server) writeAuthenticatedSession(w http.ResponseWriter, r *http.Request, status int, session authservice.Session) {
	secure := requestUsesSecureTransport(r)
	csrf, err := randomCSRFToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "无法建立安全 Web 会话")
		return
	}
	expires := time.Unix(session.ExpiresAt, 0)
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: session.Token, Path: "/", Expires: expires,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name: csrfCookieName, Value: csrf, Path: "/", Expires: expires,
		HttpOnly: false, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
	// Keep the JSON session shape for desktop/API compatibility. The browser
	// adapter deliberately discards this token and persists only the HttpOnly
	// cookie; see frontend/src/shared/api/backend.ts.
	writeJSON(w, status, session)
}

func (s *Server) clearAuthenticatedSession(w http.ResponseWriter, r *http.Request) {
	secure := requestUsesSecureTransport(r)
	for _, name := range []string{sessionCookieName, csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name: name, Value: "", Path: "/", MaxAge: -1, Expires: time.Unix(1, 0),
			HttpOnly: name == sessionCookieName, Secure: secure, SameSite: http.SameSiteStrictMode,
		})
	}
}

func validCSRFFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	header := strings.TrimSpace(r.Header.Get(csrfHeaderName))
	if header == "" || len(header) != len(cookie.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header), []byte(cookie.Value)) == 1
}

func methodRequiresCSRF(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requestOriginAllowed(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/") || !methodRequiresCSRF(r.Method) {
		return true
	}
	if site := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || strings.EqualFold(origin, "null") {
		// Non-browser clients normally omit Origin. `null` is rejected because it
		// is emitted by sandboxed/file origins and must not bootstrap/administer a
		// remotely reachable AGMP instance.
		return origin == ""
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.Path != "" && parsed.Path != "/" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	expectedScheme := "http"
	if requestUsesSecureTransport(r) {
		expectedScheme = "https"
	}
	return strings.EqualFold(parsed.Scheme, expectedScheme) && strings.EqualFold(parsed.Host, r.Host)
}

func withOriginProtection(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requestOriginAllowed(r) {
			writeError(w, http.StatusForbidden, "跨站请求已被 AGMP 安全策略拒绝")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; font-src 'self' data:")
		next.ServeHTTP(w, r)
	})
}
