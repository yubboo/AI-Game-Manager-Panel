package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type authLimitEntry struct {
	windowStart time.Time
	count       int
	lastSeen    time.Time
}
type authRateLimiter struct {
	mu      sync.Mutex
	entries map[string]authLimitEntry
	now     func() time.Time
}

func newAuthRateLimiter() *authRateLimiter {
	return &authRateLimiter{entries: map[string]authLimitEntry{}, now: time.Now}
}
func (l *authRateLimiter) allow(key string, limit int, window time.Duration) bool {
	if limit <= 0 || window <= 0 {
		return false
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.entries[key]
	if entry.windowStart.IsZero() || now.Sub(entry.windowStart) >= window {
		entry = authLimitEntry{windowStart: now}
	}
	entry.count++
	entry.lastSeen = now
	l.entries[key] = entry
	if len(l.entries) > 2048 {
		cutoff := now.Add(-2 * window)
		for candidate, value := range l.entries {
			if value.lastSeen.Before(cutoff) {
				delete(l.entries, candidate)
			}
		}
	}
	return entry.count <= limit
}
func authClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
			if parsed := net.ParseIP(realIP); parsed != nil {
				return parsed.String()
			}
		}
		if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			parts := strings.Split(forwarded, ",")
			for i := len(parts) - 1; i >= 0; i-- {
				if parsed := net.ParseIP(strings.TrimSpace(parts[i])); parsed != nil {
					return parsed.String()
				}
			}
		}
	}
	if parsed := net.ParseIP(host); parsed != nil {
		return parsed.String()
	}
	return host
}
func (s *Server) allowPublicAuth(w http.ResponseWriter, r *http.Request, action string, limit int, window time.Duration) bool {
	key := action + "|" + authClientIP(r)
	if s.authLimiter.allow(key, limit, window) {
		return true
	}
	w.Header().Set("Retry-After", "60")
	writeError(w, http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
	return false
}
