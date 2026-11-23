package media

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
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
	d := &shared.Deps{DB: database, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	res, err := database.Exec(`
		INSERT INTO users(email, password_hash, display_name, username, created_at)
		VALUES('uploader@example.test', 'hash', 'uploader', 'uploader', datetime('now'))`)
	if err != nil {
		t.Fatal(err)
	}
	userID, _ := res.LastInsertId()
	_, err = database.Exec(`
		INSERT INTO user_profiles(user_id, avatar_url, visibility, created_at)
		VALUES(?, NULL, 'public', datetime('now'))`, userID)
	if err != nil {
		t.Fatal(err)
	}
	return d, userID
}

func authCookie(t *testing.T, d *shared.Deps, userID int64) *http.Cookie {
	t.Helper()
	token := "upload-token"
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
	RegisterMedia(root, d)
	return root
}

func uploadRequest(t *testing.T, h http.Handler, filename, contentType string, data []byte, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/media", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
