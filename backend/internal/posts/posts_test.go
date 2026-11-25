package posts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func createTestPost(t *testing.T, h http.Handler, cookie *http.Cookie, body map[string]any) Post {
	t.Helper()
	rec := request(t, h, http.MethodPost, "/api/v1/posts", body, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create post: %d %s", rec.Code, rec.Body.String())
	}
	return decodeBody[Post](t, rec)
}

func TestPostCreateVoteBookmarkAndPoll(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)

	res, err := d.DB.Exec(`
		INSERT INTO media(owner_id, storage_key, mime_type, size_bytes, width, height, created_at)
		VALUES(?, 'abc.png', 'image/png', 100, 2, 2, datetime('now'))`, user)
	if err != nil {
		t.Fatal(err)
	}
	mediaID, _ := res.LastInsertId()

	post := createTestPost(t, h, cookie, map[string]any{
		"type": "image", "title": "My cosplay", "body": "A new drawing",
		"mediaIds": []int64{mediaID}, "tags": []string{"cosplay", "anime"},
		"spoiler": true, "spoilerNote": "Ending", "season": 2, "episode": 12,
	})
	if !post.Spoiler || post.SpoilerNote != "Ending" || post.Season != 2 || post.Episode != 12 {
		t.Fatalf("spoiler metadata missing: %+v", post)
	}
	if len(post.Media) != 1 || post.Media[0].URL != "/media/abc.png" {
		t.Fatalf("media = %+v", post.Media)
	}
	if len(post.Tags) != 2 || post.VoteScore != 0 || post.MyVote != 0 || post.Author.ID != user {
		t.Fatalf("post = %+v", post)
	}
	raw := request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, nil)
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw.Body.Bytes(), &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"vote_score", "my_vote", "media", "anime", "tags", "spoiler", "spoiler_note"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, raw.Body.String())
		}
	}
	if _, ok := keys["poll"]; ok {
		t.Fatalf("non-poll post must omit poll: %s", raw.Body.String())
	}

	path := fmt.Sprintf("/api/v1/posts/%d/vote", post.ID)
	voted := request(t, h, http.MethodPost, path, map[string]any{"value": 1}, cookie)
	if result := decodeBody[Post](t, voted); result.VoteScore != 1 || result.MyVote != 1 {
		t.Fatalf("upvote = %+v", result)
	}
	repeat := request(t, h, http.MethodPost, path, map[string]any{"value": 1}, cookie)
	if result := decodeBody[Post](t, repeat); result.VoteScore != 1 || result.MyVote != 1 {
		t.Fatalf("duplicate upvote = %+v", result)
	}
	alias := request(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/reactions", post.ID),
		map[string]any{"vote": -1}, cookie)
	if result := decodeBody[Post](t, alias); result.VoteScore != -1 || result.MyVote != -1 {
		t.Fatalf("reactions alias = %+v", result)
	}
	aliasKey := request(t, h, http.MethodPost, path, map[string]any{"vote": 1}, cookie)
	if result := decodeBody[Post](t, aliasKey); result.VoteScore != 1 || result.MyVote != 1 {
		t.Fatalf("vote key alias = %+v", result)
	}
	cleared := request(t, h, http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d/reactions", post.ID), nil, cookie)
	if result := decodeBody[Post](t, cleared); result.VoteScore != 0 || result.MyVote != 0 {
		t.Fatalf("delete reactions alias = %+v", result)
	}

	bookmarkPath := fmt.Sprintf("/api/v1/posts/%d/bookmark", post.ID)
	if rec := request(t, h, http.MethodPost, bookmarkPath, nil, cookie); rec.Code != http.StatusOK {
		t.Fatalf("bookmark: %d", rec.Code)
	}
	if result := decodeBody[Post](t, request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, cookie)); !result.Bookmarked {
		t.Fatal("bookmarked flag not set")
	}
	if rec := request(t, h, http.MethodDelete, bookmarkPath, nil, cookie); rec.Code != http.StatusOK {
		t.Fatalf("unbookmark: %d", rec.Code)
	}
	if result := decodeBody[Post](t, request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, cookie)); result.Bookmarked {
		t.Fatal("bookmarked flag not cleared")
	}

	poll := createTestPost(t, h, cookie, map[string]any{
		"type": "poll", "title": "Best arc?", "body": "Vote",
		"poll": map[string]any{"question": "Which arc?", "options": []string{"A", "B"}},
	})
	if poll.Poll == nil || poll.Poll.Question != "Which arc?" || len(poll.Poll.Options) != 2 || poll.Poll.TotalVotes != 0 {
		t.Fatalf("poll = %+v", poll.Poll)
	}
	optionID := poll.Poll.Options[0].ID
	votePath := fmt.Sprintf("/api/v1/posts/%d/poll-vote", poll.ID)
	for i := 0; i < 2; i++ {
		rec := request(t, h, http.MethodPost, votePath, map[string]any{"optionId": optionID}, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("poll vote: %d %s", rec.Code, rec.Body.String())
		}
	}
	result := decodeBody[Post](t, request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", poll.ID), nil, cookie))
	if result.Poll.TotalVotes != 1 || result.Poll.MyVote == nil || *result.Poll.MyVote != optionID {
		t.Fatalf("poll votes = %+v", result.Poll)
	}

	bad := request(t, h, http.MethodPost, "/api/v1/posts", map[string]any{
		"type": "poll", "title": "Bad", "poll": map[string]any{"options": []string{"only one"}},
	}, cookie)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("invalid poll: %d", bad.Code)
	}
	var envelope struct {
		Error struct {
			Details map[string]string `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bad.Body.Bytes(), &envelope); err != nil || envelope.Error.Details["poll"] == "" {
		t.Fatalf("validation details = %+v, %v", envelope, err)
	}
}

func TestCommentsNestingAndSoftDelete(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)
	post := createTestPost(t, h, cookie, map[string]any{"type": "text", "title": "Thread", "body": "Start"})

	add := func(parentID *int64, body string) *httptest.ResponseRecorder {
		payload := map[string]any{"body": body}
		if parentID != nil {
			payload["parentId"] = *parentID
		}
		return request(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), payload, cookie)
	}
	first := add(nil, "top level")
	if first.Code != http.StatusCreated {
		t.Fatalf("top comment: %d %s", first.Code, first.Body.String())
	}
	firstID := decodeBody[Comment](t, first).ID
	second := add(&firstID, "reply")
	if second.Code != http.StatusCreated {
		t.Fatalf("reply: %d %s", second.Code, second.Body.String())
	}
	secondID := decodeBody[Comment](t, second).ID
	if parent := decodeBody[Comment](t, second).ParentID; parent == nil || *parent != firstID {
		t.Fatalf("parent_id = %v", parent)
	}
	third := add(&secondID, "nested reply")
	if third.Code != http.StatusCreated {
		t.Fatalf("third level: %d %s", third.Code, third.Body.String())
	}
	thirdID := decodeBody[Comment](t, third).ID
	if tooDeep := add(&thirdID, "too deep"); tooDeep.Code != http.StatusBadRequest {
		t.Fatalf("fourth level: %d", tooDeep.Code)
	}

	otherPost := createTestPost(t, h, cookie, map[string]any{"type": "text", "title": "Other", "body": "x"})
	wrongPost := request(t, h, http.MethodPost, fmt.Sprintf("/api/v1/posts/%d/comments", otherPost.ID),
		map[string]any{"body": "wrong parent", "parentId": firstID}, cookie)
	if wrongPost.Code != http.StatusBadRequest {
		t.Fatalf("parent from another post: %d", wrongPost.Code)
	}

	list := request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), nil, nil)
	comments := decodeBody[page[Comment]](t, list)
	if len(comments.Items) != 3 {
		t.Fatalf("comments = %+v", comments)
	}
	if _, err := d.DB.Exec(`UPDATE comments SET deleted_at = datetime('now') WHERE id = ?`, firstID); err != nil {
		t.Fatal(err)
	}
	list = request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d/comments", post.ID), nil, nil)
	comments = decodeBody[page[Comment]](t, list)
	deletedComment := comments.Items[len(comments.Items)-1] // newest first
	if len(comments.Items) != 3 || deletedComment.Body != "" || !deletedComment.Deleted {
		t.Fatalf("deleted comment = %+v", comments.Items)
	}

	deleted := request(t, h, http.MethodDelete, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, cookie)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete post: %d", deleted.Code)
	}
	if rec := request(t, h, http.MethodGet, fmt.Sprintf("/api/v1/posts/%d", post.ID), nil, nil); rec.Code != http.StatusNotFound {
		t.Fatalf("deleted post GET: %d", rec.Code)
	}
	var remaining int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM comments WHERE post_id = ?`, post.ID).Scan(&remaining); err != nil || remaining != 3 {
		t.Fatalf("comments after post delete = %d, %v", remaining, err)
	}
}

func insertCommunity(t *testing.T, d *shared.Deps, owner int64, name, slug, visibility string) int64 {
	t.Helper()
	res, err := d.DB.Exec(`
		INSERT INTO communities(owner_id, name, slug, description, visibility, posting_enabled, created_at)
		VALUES(?, ?, ?, '', ?, 1, datetime('now'))`, owner, name, slug, visibility)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	_, err = d.DB.Exec(`
		INSERT INTO community_memberships(community_id, user_id, role, status, joined_at)
		VALUES(?, ?, 'owner', 'active', datetime('now'))`, id, owner)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestCommunityPrivacyAndOwnership(t *testing.T) {
	d, alice := testDeps(t)
	h := testRouter(d)
	aliceCookie := authCookie(t, d, alice)
	bob := insertUser(t, d, "bob")
	bobCookie := authCookie(t, d, bob)

	privateID := insertCommunity(t, d, alice, "Secret Club", "secret-club", "private")
	post := createTestPost(t, h, aliceCookie, map[string]any{
		"type": "text", "title": "Secret", "body": "Members only", "communitySlug": "secret-club",
	})
	if post.Community == nil || post.Community.ID != privateID || post.Community.Visibility != "private" {
		t.Fatalf("post community = %+v", post.Community)
	}
	path := fmt.Sprintf("/api/v1/posts/%d", post.ID)
	if rec := request(t, h, http.MethodGet, path, nil, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("private post as non-member: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPatch, path, map[string]any{"title": "Hacked"}, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner patch: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodDelete, path, nil, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner delete: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, path+"/comments", map[string]any{"body": "hi"}, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-member comment: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, "/api/v1/posts", map[string]any{
		"type": "text", "title": "Intruder", "communitySlug": "secret-club",
	}, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-member post: %d", rec.Code)
	}

	edited := request(t, h, http.MethodPatch, path, map[string]any{"title": "Edited title"}, aliceCookie)
	if edited.Code != http.StatusOK {
		t.Fatalf("author patch: %d %s", edited.Code, edited.Body.String())
	}
	updated := decodeBody[Post](t, edited)
	if updated.Title != "Edited title" || updated.EditedAt == "" {
		t.Fatalf("edit = %+v", updated)
	}

	// Feed and community list never leak private posts to non-members.
	feed := decodeBody[page[Post]](t, request(t, h, http.MethodGet, "/api/v1/posts", nil, nil))
	for _, item := range feed.Items {
		if item.ID == post.ID {
			t.Fatal("private post leaked into public feed")
		}
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/posts?community=secret-club", nil, bobCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("private community filter: %d", rec.Code)
	}
	memberFeed := decodeBody[page[Post]](t, request(t, h, http.MethodGet, "/api/v1/posts?community=secret-club", nil, aliceCookie))
	if len(memberFeed.Items) != 1 || memberFeed.Items[0].ID != post.ID {
		t.Fatalf("member feed = %+v", memberFeed)
	}

	if _, err := d.DB.Exec(`UPDATE communities SET posting_enabled = 0 WHERE id = ?`, privateID); err != nil {
		t.Fatal(err)
	}
	if rec := request(t, h, http.MethodPost, "/api/v1/posts", map[string]any{
		"type": "text", "title": "Nope", "communitySlug": "secret-club",
	}, aliceCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("posting disabled: %d", rec.Code)
	}
}

func TestFeedPagination(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)
	for i := 0; i < 3; i++ {
		createTestPost(t, h, cookie, map[string]any{
			"type": "text", "title": fmt.Sprintf("Post %d", i), "body": "text",
		})
	}
	first := decodeBody[page[Post]](t, request(t, h, http.MethodGet, "/api/v1/posts?limit=2", nil, nil))
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("page 1 = %+v", first)
	}
	if first.Items[0].ID <= first.Items[1].ID {
		t.Fatalf("feed must be descending: %+v", first.Items)
	}
	second := decodeBody[page[Post]](t, request(t, h, http.MethodGet,
		"/api/v1/posts?limit=2&cursor="+first.NextCursor, nil, nil))
	if len(second.Items) != 1 || second.NextCursor != "" {
		t.Fatalf("page 2 = %+v", second)
	}
}

// Stored links are rendered as href, so a non-http scheme must never be saved.
func TestPostLinkRejectsNonHTTPSchemes(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)

	for _, link := range []string{"javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "//evil.example"} {
		rec := request(t, h, http.MethodPost, "/api/v1/posts", map[string]any{
			"type": "link", "title": "A link", "linkUrl": link,
		}, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("link %q accepted: %d %s", link, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "linkUrl") {
			t.Fatalf("link %q: no field error in %s", link, rec.Body.String())
		}
	}

	post := createTestPost(t, h, cookie, map[string]any{
		"type": "link", "title": "A good link", "linkUrl": "https://example.com/art",
	})
	patched := request(t, h, http.MethodPatch, fmt.Sprintf("/api/v1/posts/%d", post.ID), map[string]any{
		"linkUrl": "javascript:alert(1)",
	}, cookie)
	if patched.Code != http.StatusBadRequest {
		t.Fatalf("patch accepted a javascript link: %d %s", patched.Code, patched.Body.String())
	}
}
