package shared

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
)

type contextKey uint8

const (
	requestIDKey contextKey = iota
	userIDKey
)

// RequestIDHeader is the header used for request correlation.
const RequestIDHeader = "X-Request-Id"

const maxRequestIDLen = 64

// RequestID echoes a client-provided X-Request-Id (sanitised and capped) or
// generates a random one, then stores it in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get(RequestIDHeader))
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(RequestIDHeader, id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom returns the request ID set by RequestID, or an empty string.
func RequestIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey).(string)
	return id
}

// Recoverer converts panics into a 500 JSON response and logs the stack trace.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger := slog.Default()
				logger.Error("panic serving request",
					"panic", fmt.Sprint(recovered),
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", RequestIDFrom(r),
					"stack", string(debug.Stack()),
				)
				WriteError(w, http.StatusInternalServerError, CodeInternalError, "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS allows a single configured origin (or *), answers preflight OPTIONS
// requests and keeps cookie-based credentials working cross-origin.
func CORS(origin string) func(http.Handler) http.Handler {
	allowed := strings.TrimSpace(origin)
	if allowed == "" {
		allowed = "http://localhost:5173"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Vary", "Origin")
			requestOrigin := r.Header.Get("Origin")
			// A wildcard never carries credentials: echoing the caller's origin
			// with Allow-Credentials would let any site read a signed-in user's
			// responses. Credentials require one exact configured origin.
			switch {
			case requestOrigin == "":
			case allowed == "*":
				w.Header().Set("Access-Control-Allow-Origin", "*")
			case requestOrigin == allowed:
				w.Header().Set("Access-Control-Allow-Origin", requestOrigin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if w.Header().Get("Access-Control-Allow-Origin") != "" {
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-Id, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth rejects requests without a valid session and puts the user ID
// in the request context for handlers.
func RequireAuth(d *Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := d.SessionUserID(r)
			if !ok {
				WriteError(w, http.StatusUnauthorized, CodeAuthRequired, "Sign in to continue.")
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth populates the user ID in context when a valid session exists,
// but always continues the request.
func OptionalAuth(d *Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if userID, ok := d.SessionUserID(r); ok {
				ctx := context.WithValue(r.Context(), userIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserID returns the authenticated user ID placed in context by RequireAuth
// or OptionalAuth.
func UserID(r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value(userIDKey).(int64)
	return userID, ok && userID > 0
}

func newRequestID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		// crypto/rand failure is exceptional; a deterministic fallback keeps
		// responses correlated with logs rather than failing the request.
		return fmt.Sprintf("fallback-%x", raw[:])
	}
	return hex.EncodeToString(raw[:])
}

func sanitizeRequestID(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > maxRequestIDLen {
		value = value[:maxRequestIDLen]
	}
	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	result := b.String()
	if result == "" || strings.Trim(result, "-") == "" {
		return ""
	}
	return result
}
