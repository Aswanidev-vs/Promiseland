package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

type searchResults struct {
	Users       []shape.UserSummary `json:"users"`
	Communities []shape.Community   `json:"communities"`
	Posts       []shape.Post        `json:"posts"`
}

func searchRouter(d *shared.Deps) chi.Router {
	r := chi.NewRouter()
	Register(r, d)
	return r
}

func doSearch(t *testing.T, r chi.Router, path, token string) (int, searchResults) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var results searchResults
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("decode search response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, results
}

func makeCommunity(t *testing.T, d *shared.Deps, ownerID int64, slug, visibility string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO communities (owner_id, name, slug, description, visibility)
		VALUES (?, ?, ?, 'description ' || ?, ?)`, ownerID, slug, slug, slug, visibility)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func makePost(t *testing.T, d *shared.Deps, authorID int64, title string, deleted bool) int64 {
	t.Helper()
	deletedAt := any(nil)
	if deleted {
		deletedAt = "2026-01-01 00:00:00"
	}
	res, err := d.DB.Exec(`
		INSERT INTO posts (author_id, type, title, body, deleted_at)
		VALUES (?, 'text', ?, 'body', ?)`, authorID, title, deletedAt)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestSearchRespectsProfileVisibilityBlocksAndPrivateCommunities(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, aliceToken := testutil.User(t, d, "alice")
	bobID, _ := testutil.User(t, d, "bob")
	carolID, _ := testutil.User(t, d, "carol")
	daveID, _ := testutil.User(t, d, "dave")

	testutil.Exec(t, d, `UPDATE user_profiles SET visibility = 'private' WHERE user_id = ?`, bobID)
	testutil.Exec(t, d, `UPDATE users SET discoverable = 0 WHERE id = ?`, carolID)
	testutil.Exec(t, d, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, aliceID, daveID)

	makeCommunity(t, d, aliceID, "public-garden", "public")
	makeCommunity(t, d, aliceID, "secret-garden", "private")

	r := searchRouter(d)
	for _, tc := range []struct {
		name  string
		query string
		token string
	}{
		{name: "private profile hidden", query: "/search?q=bob&type=users", token: ""},
		{name: "non-discoverable hidden", query: "/search?q=carol&type=users", token: ""},
		{name: "blocked user hidden", query: "/search?q=dave&type=users", token: aliceToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, results := doSearch(t, r, tc.query, tc.token)
			if status != http.StatusOK || len(results.Users) != 0 {
				t.Fatalf("status=%d users=%+v", status, results.Users)
			}
		})
	}

	status, visibleUser := doSearch(t, r, "/search?q=alice&type=users", "")
	if status != http.StatusOK || len(visibleUser.Users) != 1 || visibleUser.Users[0].ID != aliceID {
		t.Fatalf("public user search = %+v (status %d)", visibleUser, status)
	}
	status, communities := doSearch(t, r, "/search?q=garden&type=communities", "")
	if status != http.StatusOK || len(communities.Communities) != 1 || communities.Communities[0].Slug != "public-garden" {
		t.Fatalf("community search = %+v (status %d)", communities, status)
	}
}

func TestSearchEscapesWildcardsAndHidesDeletedPosts(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, _ := testutil.User(t, d, "alice")
	literal := makePost(t, d, aliceID, "100% accurate", false)
	other := makePost(t, d, aliceID, "1000 things", false)
	underscore := makePost(t, d, aliceID, "a_b", false)
	plain := makePost(t, d, aliceID, "axb", false)
	deleted := makePost(t, d, aliceID, "deleted secret", true)

	r := searchRouter(d)
	status, results := doSearch(t, r, "/search?q=100%25&type=posts", "")
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	ids := map[int64]bool{}
	for _, p := range results.Posts {
		ids[p.ID] = true
	}
	if !ids[literal] || ids[other] {
		t.Errorf("percent escape failed: got %v want only %d", ids, literal)
	}

	status, results = doSearch(t, r, "/search?q=a_b&type=posts", "")
	if status != http.StatusOK || len(results.Posts) != 1 || results.Posts[0].ID != underscore {
		t.Fatalf("underscore escape failed: %+v (status %d)", results.Posts, status)
	}
	if ids[plain] {
		t.Errorf("underscore wildcard matched unrelated post")
	}

	status, results = doSearch(t, r, "/search?q=deleted&type=posts", "")
	if status != http.StatusOK || len(results.Posts) != 0 {
		t.Fatalf("deleted post leaked: %+v", results.Posts)
	}
	_ = deleted
}

func TestSearchValidationAndResponseShape(t *testing.T) {
	d := testutil.Deps(t)
	testutil.User(t, d, "alice")
	r := searchRouter(d)
	for _, path := range []string{"/search?q=", "/search?q=" + strings.Repeat("a", 101), "/search?q=alice&type=unknown"} {
		status, _ := doSearch(t, r, path, "")
		if status != http.StatusBadRequest {
			t.Errorf("%s status = %d, want 400", path, status)
		}
	}
	status, results := doSearch(t, r, "/search?q=alice", "")
	if status != http.StatusOK || results.Users == nil || results.Communities == nil || results.Posts == nil {
		t.Fatalf("all-type response must contain all three arrays: %+v (status %d)", results, status)
	}
}
