package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
)

func testDeps(t *testing.T) *shared.Deps {
	t.Helper()
	database, _ := testutil.TempDB(t, "router.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return &shared.Deps{
		DB:     database,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: config.Config{CORSOrigin: "http://localhost:5173"},
	}
}

func TestHealthzAndReadyz(t *testing.T) {
	handler := New(testDeps(t))

	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantStatusValue string
	}{
		{name: "healthz", path: "/healthz", wantStatus: http.StatusOK, wantStatusValue: "ok"},
		{name: "readyz", path: "/readyz", wantStatus: http.StatusOK, wantStatusValue: "ready"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			var payload map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode body: %v (%q)", err, rec.Body.String())
			}
			if len(payload) != 1 || payload["status"] != tt.wantStatusValue {
				t.Errorf("body = %v", payload)
			}
			if strings.TrimSpace(rec.Body.String()) != `{"status":"`+tt.wantStatusValue+`"}` {
				t.Errorf("non-canonical body = %q", rec.Body.String())
			}
		})
	}
}

func TestReadyzWhenDBUnavailable(t *testing.T) {
	deps := testDeps(t)
	// A closed DB simulates a failed database connection.
	if err := deps.DB.Close(); err != nil {
		t.Fatalf("close DB: %v", err)
	}

	rec := httptest.NewRecorder()
	New(deps).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if payload.Error.Code != shared.CodeDBUnavailable {
		t.Errorf("code = %q", payload.Error.Code)
	}
}

func TestUnknownRoutesReturnJSON404(t *testing.T) {
	handler := New(testDeps(t))
	for _, path := range []string{"/api/v1/does-not-exist", "/api/v1/", "/nope"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if payload.Error.Code != shared.CodeNotFound {
				t.Errorf("code = %q", payload.Error.Code)
			}
		})
	}
}

func TestRequestIDAndCORS(t *testing.T) {
	handler := New(testDeps(t))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if id := rec.Header().Get(shared.RequestIDHeader); len(id) != 32 {
		t.Errorf("generated request id = %q", id)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(shared.RequestIDHeader, "client-supplied-123")
	handler.ServeHTTP(rec, req)
	if got := rec.Header().Get(shared.RequestIDHeader); got != "client-supplied-123" {
		t.Errorf("echoed request id = %q", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodOptions, "/api/v1/auth/register", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("allow origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("allow credentials = %q", got)
	}
}

func TestRequireAuthMiddleware(t *testing.T) {
	deps := testDeps(t)
	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := shared.UserID(r)
		if !ok {
			t.Error("UserID missing from context")
			return
		}
		shared.WriteJSON(w, http.StatusOK, map[string]int64{"user_id": id})
	})
	handler := shared.RequireAuth(deps)(protected)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want 401", rec.Code)
	}

	// Seed a user and a live session, then authenticate with the cookie.
	if _, err := deps.DB.Exec(`
		INSERT INTO users (id, email, password_hash, display_name)
		VALUES (1, 'auth@example.com', 'hash', 'Auth User')`); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	sum := sha256.Sum256([]byte("valid-token"))
	if _, err := deps.DB.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, 1, datetime('now', '+1 day'))`, hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: "valid-token"})
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d (%s)", rec.Code, rec.Body.String())
	}
}
