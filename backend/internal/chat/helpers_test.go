package chat

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func testDeps(t *testing.T) (*shared.Deps, int64) {
	t.Helper()
	database, _ := testutil.TempDB(t, "chat-test.db")
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
	if _, err := d.DB.Exec(`
		INSERT INTO user_profiles(user_id, avatar_url, visibility, created_at)
		VALUES(?, NULL, 'public', datetime('now'))`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

func authCookie(t *testing.T, d *shared.Deps, userID int64) *http.Cookie {
	t.Helper()
	token := fmt.Sprintf("chat-test-token-%d", userID)
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

type apiErrorEnvelope struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Details map[string]string `json:"details"`
	} `json:"error"`
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	return decodeBody[apiErrorEnvelope](t, rec).Error.Code
}

func itoa(id int64) string {
	return strconv.FormatInt(id, 10)
}

// withClock replaces the rate-limit clock for the duration of a test and
// returns an advancer, keeping rate-limit assertions deterministic.
func withClock(t *testing.T) func(time.Duration) {
	t.Helper()
	current := time.Now()
	nowFn = func() time.Time { return current }
	t.Cleanup(func() { nowFn = time.Now })
	return func(d time.Duration) { current = current.Add(d) }
}

func createServer(t *testing.T, h http.Handler, cookie *http.Cookie, name string) ChatServer {
	t.Helper()
	rec := request(t, h, http.MethodPost, "/api/v1/chat/servers", map[string]any{
		"name": name, "description": "A test server",
	}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create server: status %d body %q", rec.Code, rec.Body.String())
	}
	return decodeBody[ChatServer](t, rec)
}

func createDirect(t *testing.T, h http.Handler, cookie *http.Cookie, otherID int64) Conversation {
	t.Helper()
	rec := request(t, h, http.MethodPost, "/api/v1/chat/conversations",
		map[string]any{"memberIds": []int64{otherID}}, cookie)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("create direct conversation: status %d body %q", rec.Code, rec.Body.String())
	}
	return decodeBody[Conversation](t, rec)
}

func firstChannelID(t *testing.T, d *shared.Deps, serverID int64) int64 {
	t.Helper()
	var id int64
	if err := d.DB.QueryRow(`SELECT id FROM chat_channels WHERE server_id = ? ORDER BY id LIMIT 1`, serverID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func sendChannelMessage(t *testing.T, h http.Handler, cookie *http.Cookie, channelID int64, body string) ChatMessage {
	t.Helper()
	rec := request(t, h, http.MethodPost,
		"/api/v1/chat/channels/"+itoa(channelID)+"/messages",
		map[string]any{"body": body}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send message: status %d body %q", rec.Code, rec.Body.String())
	}
	return decodeBody[ChatMessage](t, rec)
}

func sendConversationMessage(t *testing.T, h http.Handler, cookie *http.Cookie, conversationID int64, body string) ChatMessage {
	t.Helper()
	rec := request(t, h, http.MethodPost,
		"/api/v1/chat/conversations/"+itoa(conversationID)+"/messages",
		map[string]any{"body": body}, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send conversation message: status %d body %q", rec.Code, rec.Body.String())
	}
	return decodeBody[ChatMessage](t, rec)
}
