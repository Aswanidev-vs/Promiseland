package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

func testRouter(d *shared.Deps) chi.Router {
	r := chi.NewRouter()
	r.Group(func(g chi.Router) {
		g.Use(shared.RequireAuth(d))
		Register(g, d)
	})
	return r
}

func call(t *testing.T, r chi.Router, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func grantStaff(t *testing.T, d *shared.Deps, userID int64) {
	t.Helper()
	testutil.Exec(t, d, `INSERT OR IGNORE INTO platform_moderators (user_id) VALUES (?)`, userID)
}

func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %s: %v", string(body), err)
	}
	return value
}

func TestConsoleRefusesOrdinaryAccounts(t *testing.T) {
	d := testutil.Deps(t)
	_, token := testutil.User(t, d, "member")
	r := testRouter(d)

	if rec := call(t, r, http.MethodGet, "/admin/overview", token, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("overview without a grant = %d, want 403", rec.Code)
	}
	if rec := call(t, r, http.MethodGet, "/admin/users", token, ""); rec.Code != http.StatusForbidden {
		t.Fatalf("users without a grant = %d, want 403", rec.Code)
	}

	rec := call(t, r, http.MethodGet, "/admin/me", token, "")
	if rec.Code != http.StatusOK || decode[staffReply](t, rec.Body.Bytes()).IsStaff {
		t.Fatalf("me for a member = %d %s", rec.Code, rec.Body.String())
	}
}

func TestOverviewAndTablesForStaff(t *testing.T) {
	d := testutil.Deps(t)
	staffID, staffToken := testutil.User(t, d, "keeper")
	authorID, _ := testutil.User(t, d, "author")
	grantStaff(t, d, staffID)

	testutil.Exec(t, d, `
		INSERT INTO posts (author_id, type, title, body) VALUES (?, 'text', 'First post', 'hello')`,
		authorID, authorID)
	testutil.Exec(t, d, `
		INSERT INTO communities (owner_id, name, slug, description) VALUES (?, 'Craft', 'craft', 'props')`,
		authorID)
	r := testRouter(d)

	rec := call(t, r, http.MethodGet, "/admin/me", staffToken, "")
	if !decode[staffReply](t, rec.Body.Bytes()).IsStaff {
		t.Fatal("staff flag should be true for a granted account")
	}

	rec = call(t, r, http.MethodGet, "/admin/overview", staffToken, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("overview = %d %s", rec.Code, rec.Body.String())
	}
	overview := decode[overviewReply](t, rec.Body.Bytes())
	if overview.Counts.Users != 2 || overview.Counts.Posts != 1 || overview.Counts.Communities != 1 {
		t.Fatalf("counts = %+v", overview.Counts)
	}
	if len(overview.RecentPosts) != 1 || overview.RecentPosts[0].Title != "First post" {
		t.Fatalf("recent posts = %+v", overview.RecentPosts)
	}
	if len(overview.RecentUsers) != 2 {
		t.Fatalf("recent users = %d", len(overview.RecentUsers))
	}

	users := decode[usersReply](t, call(t, r, http.MethodGet, "/admin/users", staffToken, "").Body.Bytes())
	if len(users.Items) != 2 {
		t.Fatalf("users = %d", len(users.Items))
	}
	if !users.Items[1].IsStaff && !users.Items[0].IsStaff {
		t.Fatal("the granted account should be flagged as staff")
	}

	filtered := decode[usersReply](t, call(t, r, http.MethodGet, "/admin/users?q=author", staffToken, "").Body.Bytes())
	if len(filtered.Items) != 1 || filtered.Items[0].Username != "author" {
		t.Fatalf("search = %+v", filtered.Items)
	}

	posts := decode[postsReply](t, call(t, r, http.MethodGet, "/admin/posts", staffToken, "").Body.Bytes())
	if len(posts.Items) != 1 || posts.Items[0].Author != "author" {
		t.Fatalf("posts = %+v", posts.Items)
	}

	communities := decode[communitiesReply](t, call(t, r, http.MethodGet, "/admin/communities", staffToken, "").Body.Bytes())
	if len(communities.Items) != 1 || communities.Items[0].Slug != "craft" {
		t.Fatalf("communities = %+v", communities.Items)
	}
}

func TestSuspendEndsAccessAndSelfSuspendIsRefused(t *testing.T) {
	d := testutil.Deps(t)
	staffID, staffToken := testutil.User(t, d, "keeper")
	memberID, memberToken := testutil.User(t, d, "member")
	grantStaff(t, d, staffID)
	r := testRouter(d)

	if rec := call(t, r, http.MethodPatch, "/admin/users/"+strconv.FormatInt(staffID, 10), staffToken,
		`{"is_active":false}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("self-suspend = %d, want 400", rec.Code)
	}

	rec := call(t, r, http.MethodPatch, "/admin/users/"+strconv.FormatInt(memberID, 10), staffToken,
		`{"is_active":false}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend = %d %s", rec.Code, rec.Body.String())
	}
	state := decode[userStateReply](t, rec.Body.Bytes())
	if state.IsActive {
		t.Fatal("suspended account should report is_active false")
	}

	var active int
	if err := d.DB.QueryRow(`SELECT is_active FROM users WHERE id = ?`, memberID).Scan(&active); err != nil {
		t.Fatalf("read user: %v", err)
	}
	if active != 0 {
		t.Fatal("user row was not suspended")
	}
	var sessions int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id = ?`, memberID).Scan(&sessions); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("sessions left = %d, want 0", sessions)
	}
	// The old cookie no longer authenticates.
	if rec := call(t, r, http.MethodGet, "/admin/me", memberToken, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("suspended session = %d, want 401", rec.Code)
	}

	// Reinstating is the same endpoint with true.
	if rec := call(t, r, http.MethodPatch, "/admin/users/"+strconv.FormatInt(memberID, 10), staffToken,
		`{"is_active":true}`); rec.Code != http.StatusOK {
		t.Fatalf("reinstate = %d", rec.Code)
	}
}
