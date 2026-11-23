package feed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

type feedPage struct {
	Items      []shape.Post `json:"items"`
	NextCursor string       `json:"next_cursor"`
}

func feedRouter(d *shared.Deps) chi.Router {
	r := chi.NewRouter()
	Register(r, d)
	return r
}

func getFeed(t *testing.T, r chi.Router, path, token string) (int, feedPage) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: token})
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var page feedPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode feed response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, page
}

func insertCommunity(t *testing.T, d *shared.Deps, ownerID int64, slug, visibility string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO communities (owner_id, name, slug, description, visibility)
		VALUES (?, ?, ?, '', ?)`, ownerID, slug, slug, visibility)
	if err != nil {
		t.Fatalf("insert community: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func insertPost(t *testing.T, d *shared.Deps, authorID int64, communityID *int64, title string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO posts (author_id, community_id, type, title, body)
		VALUES (?, ?, 'text', ?, 'body')`, authorID, communityID, title)
	if err != nil {
		t.Fatalf("insert post: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestFeedExcludesPrivateAndBlockedPosts(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, aliceToken := testutil.User(t, d, "alice")
	bobID, _ := testutil.User(t, d, "bob")
	carolID, _ := testutil.User(t, d, "carol")

	public := insertCommunity(t, d, aliceID, "public-place", "public")
	// Bob owns the private community so Alice is neither owner nor member.
	private := insertCommunity(t, d, bobID, "private-place", "private")
	// Bob is an active member of the private community.
	testutil.Exec(t, d, `INSERT INTO community_memberships (community_id, user_id, role, status)
		VALUES (?, ?, 'member', 'active')`, private, bobID)
	// Alice blocks Carol.
	testutil.Exec(t, d, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, aliceID, carolID)

	visiblePublic := insertPost(t, d, bobID, &public, "public by bob")
	hiddenPrivate := insertPost(t, d, carolID, &private, "private by carol")
	hiddenBlocked := insertPost(t, d, carolID, &public, "public by blocked carol")
	memberPrivate := insertPost(t, d, bobID, &private, "private by member bob")

	cases := []struct {
		name      string
		token     string
		wantIDs   map[int64]bool
		forbidden map[int64]bool
	}{
		{name: "signed-in alice", token: aliceToken, wantIDs: map[int64]bool{visiblePublic: true}, forbidden: map[int64]bool{hiddenPrivate: true, hiddenBlocked: true, memberPrivate: true}},
		{name: "anonymous", token: "", wantIDs: map[int64]bool{visiblePublic: true, hiddenBlocked: true}, forbidden: map[int64]bool{hiddenPrivate: true, memberPrivate: true}},
	}
	r := feedRouter(d)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, page := getFeed(t, r, "/feed?limit=50", tc.token)
			if status != http.StatusOK {
				t.Fatalf("status = %d", status)
			}
			got := map[int64]bool{}
			for _, p := range page.Items {
				got[p.ID] = true
			}
			for id := range tc.wantIDs {
				if !got[id] {
					t.Errorf("missing post %d", id)
				}
			}
			for id := range tc.forbidden {
				if got[id] {
					t.Errorf("unexpected post %d", id)
				}
			}
		})
	}
}

func TestFeedScopesAndInterests(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, aliceToken := testutil.User(t, d, "alice")
	bobID, _ := testutil.User(t, d, "bob")
	carolID, _ := testutil.User(t, d, "carol")
	public := insertCommunity(t, d, aliceID, "public-place", "public")

	testutil.Exec(t, d, `INSERT INTO follows (follower_id, followed_id) VALUES (?, ?)`, aliceID, bobID)
	bobPost := insertPost(t, d, bobID, &public, "bob followed")
	carolPost := insertPost(t, d, carolID, &public, "carol not followed")
	testutil.Exec(t, d, `INSERT INTO post_tags (post_id, tag) VALUES (?, 'cosplay')`, carolPost)
	testutil.Exec(t, d, `INSERT OR IGNORE INTO interests (name, category) VALUES ('cosplay', 'creative')`)
	var interestID int64
	if err := d.DB.QueryRow(`SELECT id FROM interests WHERE name = 'cosplay'`).Scan(&interestID); err != nil {
		t.Fatal(err)
	}
	testutil.Exec(t, d, `INSERT INTO user_interests (user_id, interest_id) VALUES (?, ?)`, aliceID, interestID)

	r := feedRouter(d)
	status, following := getFeed(t, r, "/feed?scope=following&limit=50", aliceToken)
	if status != http.StatusOK || len(following.Items) != 1 || following.Items[0].ID != bobPost {
		t.Fatalf("following feed = %+v (status %d)", following, status)
	}
	status, forYou := getFeed(t, r, "/feed?scope=for-you&limit=50", aliceToken)
	if status != http.StatusOK {
		t.Fatalf("for-you status = %d", status)
	}
	ids := map[int64]bool{}
	for _, p := range forYou.Items {
		ids[p.ID] = true
	}
	if !ids[bobPost] || !ids[carolPost] {
		t.Errorf("for-you should include followed and interest-tagged posts, got %v", ids)
	}
	if status, _ = getFeed(t, r, "/feed?scope=invalid", aliceToken); status != http.StatusBadRequest {
		t.Errorf("invalid scope status = %d", status)
	}
}

func TestFeedExcludesSuspendedAuthors(t *testing.T) {
	d := testutil.Deps(t)
	viewerID, viewerToken := testutil.User(t, d, "viewer")
	authorID, _ := testutil.User(t, d, "author")
	public := insertCommunity(t, d, viewerID, "public-place", "public")
	post := insertPost(t, d, authorID, &public, "before suspension")

	r := feedRouter(d)
	_, page := getFeed(t, r, "/feed?limit=50", viewerToken)
	found := false
	for _, item := range page.Items {
		if item.ID == post {
			found = true
		}
	}
	if !found {
		t.Fatal("post should be visible before suspension")
	}
	testutil.Exec(t, d, `
		INSERT INTO suspensions (user_id, reason, is_active)
		VALUES (?, 'abuse', 1)`, authorID)
	_, page = getFeed(t, r, "/feed?limit=50", viewerToken)
	for _, item := range page.Items {
		if item.ID == post {
			t.Fatal("suspended author's post must be excluded from the feed")
		}
	}
}

func TestFeedPostShapeAndPagination(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, aliceToken := testutil.User(t, d, "alice")
	public := insertCommunity(t, d, aliceID, "public-place", "public")
	first := insertPost(t, d, aliceID, &public, "first")
	second := insertPost(t, d, aliceID, &public, "second")
	third := insertPost(t, d, aliceID, &public, "third")

	r := feedRouter(d)
	status, page := getFeed(t, r, "/feed?limit=2", aliceToken)
	if status != http.StatusOK || len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("first page = %+v (status %d)", page, status)
	}
	if page.Items[0].ID != third || page.Items[1].ID != second {
		t.Errorf("expected newest-first order, got %d, %d", page.Items[0].ID, page.Items[1].ID)
	}
	p := page.Items[0]
	if p.Author.ID != aliceID || p.Community == nil || p.Community.Slug != "public-place" {
		t.Errorf("author/community not expanded: %+v", p)
	}
	if p.Media == nil || p.Anime == nil || p.Tags == nil {
		t.Errorf("array fields must not be null: %+v", p)
	}
	status, last := getFeed(t, r, "/feed?limit=2&cursor="+page.NextCursor, aliceToken)
	if status != http.StatusOK || len(last.Items) != 1 || last.Items[0].ID != first || last.NextCursor != "" {
		t.Fatalf("last page = %+v (status %d)", last, status)
	}
}
