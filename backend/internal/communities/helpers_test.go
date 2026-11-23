package communities

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func testDeps(t *testing.T) (*shared.Deps, int64) {
	t.Helper()
	database, _ := testutil.TempDB(t, "test.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatal(err)
	}
	d := &shared.Deps{
		DB:  database,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return d, insertUser(t, d, "alice")
}

func insertUser(t *testing.T, d *shared.Deps, name string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO users(email, password_hash, display_name, username, created_at)
		VALUES(?, 'hash', ?, ?, datetime('now'))`, name+"@example.test", name, name)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	_, err = d.DB.Exec(`
		INSERT INTO user_profiles(user_id, avatar_url, visibility, created_at)
		VALUES(?, NULL, 'public', datetime('now'))`, id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func authCookie(t *testing.T, d *shared.Deps, userID int64) *http.Cookie {
	t.Helper()
	token := "test-token-" + string(rune('A'+userID))
	sum := sha256.Sum256([]byte(token))
	_, err := d.DB.Exec(`
		INSERT INTO sessions(token_hash, user_id, expires_at, created_at)
		VALUES(?, ?, datetime('now','+1 day'), datetime('now'))`,
		hex.EncodeToString(sum[:]), userID)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: shared.SessionCookieName, Value: token}
}

func testRouter(d *shared.Deps) http.Handler {
	root := chi.NewRouter()
	root.Use(shared.OptionalAuth(d))
	root.Route("/api/v1", func(r chi.Router) {
		Register(r, d)
		RegisterProtected(r.With(shared.RequireAuth(d)), d)
	})
	return root
}

func request(t *testing.T, h http.Handler, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return value
}
