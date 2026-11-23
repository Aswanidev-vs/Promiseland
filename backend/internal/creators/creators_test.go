package creators

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

type testEnv struct {
	d *shared.Deps
	h http.Handler
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	database, _ := testutil.TempDB(t, "creators.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatal(err)
	}
	d := &shared.Deps{DB: database, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	root := chi.NewRouter()
	root.Use(shared.OptionalAuth(d))
	root.Route("/api/v1", func(r chi.Router) {
		Register(r, d)
		RegisterProtected(r.With(shared.RequireAuth(d)), d)
	})
	return &testEnv{d: d, h: root}
}

func (e *testEnv) user(t *testing.T, username, visibility string) (int64, *http.Cookie) {
	t.Helper()
	res, err := e.d.DB.Exec(`
		INSERT INTO users(email, password_hash, display_name, username)
		VALUES (?, 'hash', ?, ?)`, username+"@example.test", username, username)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	if _, err := e.d.DB.Exec(`INSERT INTO user_profiles(user_id, visibility) VALUES (?, ?)`, id, visibility); err != nil {
		t.Fatal(err)
	}
	token := "creators-" + username
	sum := sha256.Sum256([]byte(token))
	if _, err := e.d.DB.Exec(`
		INSERT INTO sessions(token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+1 day'))`, hex.EncodeToString(sum[:]), id); err != nil {
		t.Fatal(err)
	}
	return id, &http.Cookie{Name: shared.SessionCookieName, Value: token}
}

func (e *testEnv) post(t *testing.T, authorID int64, category string) int64 {
	t.Helper()
	res, err := e.d.DB.Exec(`
		INSERT INTO posts(author_id, type, title, body, creator_category)
		VALUES (?, 'image', ?, 'body', ?)`, authorID, "work-"+category, category)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func (e *testEnv) do(t *testing.T, method, path string, body any, cookie *http.Cookie) *httptest.ResponseRecorder {
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
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error %q: %v", rec.Body.String(), err)
	}
	return payload.Error.Code
}

func TestFeaturedToggleLimitAndOwnership(t *testing.T) {
	e := newTestEnv(t)
	alice, aliceCookie := e.user(t, "alice", "public")
	bob, _ := e.user(t, "bob", "public")

	posts := make([]int64, 7)
	for i := range posts {
		posts[i] = e.post(t, alice, "cosplay")
	}
	foreignPost := e.post(t, bob, "fan_art")

	for i := 0; i < maxFeatured; i++ {
		rec := e.do(t, http.MethodPost, "/api/v1/users/me/featured/"+itoa(posts[i]), nil, aliceCookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("feature %d = %d: %s", i, rec.Code, rec.Body.String())
		}
	}
	over := e.do(t, http.MethodPost, "/api/v1/users/me/featured/"+itoa(posts[6]), nil, aliceCookie)
	if over.Code != http.StatusConflict || errorCode(t, over) != codeLimit {
		t.Fatalf("over-limit = %d %q", over.Code, errorCode(t, over))
	}
	duplicate := e.do(t, http.MethodPost, "/api/v1/users/me/featured/"+itoa(posts[0]), nil, aliceCookie)
	if duplicate.Code != http.StatusOK {
		t.Errorf("duplicate feature = %d", duplicate.Code)
	}
	foreign := e.do(t, http.MethodPost, "/api/v1/users/me/featured/"+itoa(foreignPost), nil, aliceCookie)
	if foreign.Code != http.StatusForbidden || errorCode(t, foreign) != codeForbidden {
		t.Errorf("foreign feature = %d %q", foreign.Code, errorCode(t, foreign))
	}
	unfeature := e.do(t, http.MethodDelete, "/api/v1/users/me/featured/"+itoa(posts[0]), nil, aliceCookie)
	if unfeature.Code != http.StatusOK {
		t.Errorf("unfeature = %d", unfeature.Code)
	}
	after := e.do(t, http.MethodPost, "/api/v1/users/me/featured/"+itoa(posts[6]), nil, aliceCookie)
	if after.Code != http.StatusOK {
		t.Errorf("feature after removal = %d", after.Code)
	}

	detail := e.do(t, http.MethodGet, "/api/v1/creators/alice", nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("creator detail = %d: %s", detail.Code, detail.Body.String())
	}
	var creator Creator
	if err := json.Unmarshal(detail.Body.Bytes(), &creator); err != nil {
		t.Fatal(err)
	}
	if len(creator.FeaturedPosts) != maxFeatured {
		t.Errorf("featured posts = %d, want %d", len(creator.FeaturedPosts), maxFeatured)
	}
	if creator.FeaturedPosts[0].ID != posts[1] {
		t.Errorf("featured order first = %d, want %d", creator.FeaturedPosts[0].ID, posts[1])
	}
	if _, ok := detailBodyKeys(t, detail)["creator_category"]; !ok {
		t.Error("featured post JSON missing creator_category")
	}
}

func detailBodyKeys(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		FeaturedPosts []map[string]any `json:"featured_posts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.FeaturedPosts) == 0 {
		return map[string]any{}
	}
	return payload.FeaturedPosts[0]
}

func TestCreatorListingFiltersAndPrivacy(t *testing.T) {
	e := newTestEnv(t)
	alice, aliceCookie := e.user(t, "alice", "public")
	bob, _ := e.user(t, "bob", "public")
	privateUser, privateCookie := e.user(t, "privateone", "private")
	blockedID, _ := e.user(t, "blocked", "public")

	e.post(t, alice, "cosplay")
	e.post(t, bob, "fan_art")
	e.post(t, privateUser, "cosplay")
	e.post(t, blockedID, "cosplay")
	if _, err := e.d.DB.Exec(`INSERT INTO blocks(blocker_id, blocked_id) VALUES (?, ?)`, alice, blockedID); err != nil {
		t.Fatal(err)
	}

	var all page[Creator]
	rec := e.do(t, http.MethodGet, "/api/v1/creators?limit=50", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &all); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, c := range all.Items {
		names[c.User.Username] = true
	}
	if !names["alice"] || !names["bob"] || names["privateone"] || !names["blocked"] {
		t.Errorf("unexpected creator list = %v", names)
	}

	// Alice blocked the "blocked" creator, so it disappears for her.
	asAlice := e.do(t, http.MethodGet, "/api/v1/creators?limit=50", nil, aliceCookie)
	var aliceView page[Creator]
	if err := json.Unmarshal(asAlice.Body.Bytes(), &aliceView); err != nil {
		t.Fatal(err)
	}
	for _, c := range aliceView.Items {
		if c.User.Username == "blocked" {
			t.Error("blocked creator should be hidden from the blocking viewer")
		}
	}

	category := e.do(t, http.MethodGet, "/api/v1/creators?category=fan_art", nil, nil)
	var fanArt page[Creator]
	if err := json.Unmarshal(category.Body.Bytes(), &fanArt); err != nil {
		t.Fatal(err)
	}
	if len(fanArt.Items) != 1 || fanArt.Items[0].User.Username != "bob" {
		t.Errorf("category filter = %+v", fanArt.Items)
	}

	search := e.do(t, http.MethodGet, "/api/v1/creators?q=ali", nil, nil)
	var found page[Creator]
	if err := json.Unmarshal(search.Body.Bytes(), &found); err != nil {
		t.Fatal(err)
	}
	if len(found.Items) != 1 || found.Items[0].User.Username != "alice" {
		t.Errorf("q filter = %+v", found.Items)
	}

	asPrivate := e.do(t, http.MethodGet, "/api/v1/creators?limit=50", nil, privateCookie)
	var privatePage page[Creator]
	if err := json.Unmarshal(asPrivate.Body.Bytes(), &privatePage); err != nil {
		t.Fatal(err)
	}
	selfVisible := false
	for _, c := range privatePage.Items {
		if c.User.Username == "privateone" {
			selfVisible = true
		}
	}
	if !selfVisible {
		t.Error("private profile owner should see themselves in creator discovery")
	}

	first := e.do(t, http.MethodGet, "/api/v1/creators?limit=1", nil, nil)
	var paged page[Creator]
	if err := json.Unmarshal(first.Body.Bytes(), &paged); err != nil {
		t.Fatal(err)
	}
	if len(paged.Items) != 1 || paged.NextCursor == "" {
		t.Errorf("pagination = %+v", paged)
	}
	second := e.do(t, http.MethodGet, "/api/v1/creators?limit=1&cursor="+paged.NextCursor, nil, nil)
	var paged2 page[Creator]
	if err := json.Unmarshal(second.Body.Bytes(), &paged2); err != nil {
		t.Fatal(err)
	}
	if len(paged2.Items) != 1 || paged2.Items[0].User.ID == paged.Items[0].User.ID {
		t.Errorf("second page = %+v", paged2.Items)
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
