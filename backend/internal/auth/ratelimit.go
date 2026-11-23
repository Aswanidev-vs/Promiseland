package auth

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const rateLimitWindow = 15 * time.Minute

// rateLimiter is a small, thread-safe in-memory fixed-window limiter. State is
// intentionally process-local: restarting the process clears it, which is
// acceptable for the MVP's single-instance deployment.
type rateLimiter struct {
	mu    sync.Mutex
	hits  map[string]*rateWindow
	count int
}

type rateWindow struct {
	count   int
	resetAt time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{hits: make(map[string]*rateWindow)}
}

func (l *rateLimiter) allow(key string, limit int) (allowed bool, retryAfter time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	// Periodic sweep keeps the map from growing without bound.
	l.count++
	if l.count%256 == 0 {
		for k, w := range l.hits {
			if now.After(w.resetAt) {
				delete(l.hits, k)
			}
		}
	}

	window, ok := l.hits[key]
	if !ok || !now.Before(window.resetAt) {
		l.hits[key] = &rateWindow{count: 1, resetAt: now.Add(rateLimitWindow)}
		return true, 0
	}
	if window.count >= limit {
		return false, time.Until(window.resetAt)
	}
	window.count++
	return true, 0
}

func (l *rateLimiter) middleware(route string, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, retryAfter := l.allow(clientIP(r)+"|"+route, limit)
			if !allowed {
				seconds := int(retryAfter / time.Second)
				if seconds < 1 {
					seconds = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				shared.WriteError(w, http.StatusTooManyRequests, codeRateLimited,
					"Too many attempts. Please try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP uses the connection address only. Forwarded headers are easy to
// spoof and are deliberately ignored for rate-limit keys.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
