package communities

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateCommunityRulesAndVisibility(t *testing.T) {
	d, owner := testDeps(t)
	h := testRouter(d)
	ownerCookie := authCookie(t, d, owner)
	other := insertUser(t, d, "bob")
	otherCookie := authCookie(t, d, other)

	created := request(t, h, http.MethodPost, "/api/v1/communities", map[string]any{
		"name": "Attack on Titan", "slug": "attack-on-titan", "description": "Discuss the series",
		"visibility": "private",
	}, ownerCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	community := decodeBody[Community](t, created)
	if community.Visibility != "private" || !community.IsMember || community.Role == nil || *community.Role != "owner" {
		t.Fatalf("unexpected community: %+v", community)
	}

	rulesRec := request(t, h, http.MethodGet, "/api/v1/communities/attack-on-titan/rules", nil, nil)
	if rulesRec.Code != http.StatusForbidden {
		t.Fatalf("private rules as anon: %d", rulesRec.Code)
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/communities/attack-on-titan", nil, otherCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("private community as non-member: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/communities/attack-on-titan", nil, ownerCookie); rec.Code != http.StatusOK {
		t.Fatalf("private community as owner: %d", rec.Code)
	}

	joined := request(t, h, http.MethodPost, "/api/v1/communities/attack-on-titan/join", nil, otherCookie)
	if joined.Code != http.StatusOK {
		t.Fatalf("join private: %d %s", joined.Code, joined.Body.String())
	}
	if status := decodeBody[map[string]string](t, joined)["status"]; status != "pending" {
		t.Fatalf("join status = %q", status)
	}
	if rec := request(t, h, http.MethodGet, "/api/v1/communities/attack-on-titan", nil, otherCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("pending member must not see private community: %d", rec.Code)
	}

	// Only the owner can change settings.
	if rec := request(t, h, http.MethodPatch, "/api/v1/communities/attack-on-titan",
		map[string]any{"name": "Hijacked"}, otherCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner patch: %d", rec.Code)
	}
	patched := request(t, h, http.MethodPatch, "/api/v1/communities/attack-on-titan",
		map[string]any{"name": "Titan Talk", "description": "Updated", "postingEnabled": false}, ownerCookie)
	if patched.Code != http.StatusOK {
		t.Fatalf("owner patch: %d %s", patched.Code, patched.Body.String())
	}
	updated := decodeBody[Community](t, patched)
	if updated.Name != "Titan Talk" || updated.Description != "Updated" {
		t.Fatalf("patch not applied: %+v", updated)
	}
	var posting int
	if err := d.DB.QueryRow(`SELECT posting_enabled FROM communities WHERE slug = 'attack-on-titan'`).Scan(&posting); err != nil || posting != 0 {
		t.Fatalf("posting_enabled = %d, err %v", posting, err)
	}

	var ruleCount int
	if err := d.DB.QueryRow(`SELECT COUNT(*) FROM community_rules`).Scan(&ruleCount); err != nil || ruleCount != 2 {
		t.Fatalf("default rules = %d, err %v", ruleCount, err)
	}
	rules := decodeBody[[]CommunityRule](t, request(t, h, http.MethodGet, "/api/v1/communities/attack-on-titan/rules", nil, ownerCookie))
	if len(rules) != 2 {
		t.Fatalf("rules response = %+v", rules)
	}

	// Duplicate slugs are rejected with the documented code.
	dup := request(t, h, http.MethodPost, "/api/v1/communities", map[string]any{
		"name": "Another", "slug": "attack-on-titan",
	}, otherCookie)
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate slug: %d %s", dup.Code, dup.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(dup.Body.Bytes(), &envelope); err != nil || envelope.Error.Code != "slug_taken" {
		t.Fatalf("duplicate slug error = %+v, %v", envelope, err)
	}
}

func TestPublicJoinLeaveAndOwnerCannotLeave(t *testing.T) {
	d, owner := testDeps(t)
	h := testRouter(d)
	ownerCookie := authCookie(t, d, owner)
	member := insertUser(t, d, "carol")
	memberCookie := authCookie(t, d, member)

	created := request(t, h, http.MethodPost, "/api/v1/communities", map[string]any{
		"name": "Public Space", "slug": "public-space", "visibility": "public",
	}, ownerCookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	publicCommunityID := decodeBody[Community](t, created).ID
	list := request(t, h, http.MethodGet, "/api/v1/communities", nil, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("list: %d %s", list.Code, list.Body.String())
	}
	listPage := decodeBody[page[Community]](t, list)
	if len(listPage.Items) != 1 || listPage.Items[0].Slug != "public-space" {
		t.Fatalf("public list: %+v", listPage)
	}

	if rec := request(t, h, http.MethodPost, "/api/v1/communities/public-space/join", nil, memberCookie); rec.Code != http.StatusOK {
		t.Fatalf("join public: %d", rec.Code)
	}
	members := request(t, h, http.MethodGet, "/api/v1/communities/public-space/members", nil, nil)
	if members.Code != http.StatusOK {
		t.Fatalf("members: %d", members.Code)
	}
	if got := decodeBody[page[UserSummary]](t, members); len(got.Items) != 2 {
		t.Fatalf("members = %+v", got)
	}
	if rec := request(t, h, http.MethodDelete, "/api/v1/communities/public-space/membership", nil, memberCookie); rec.Code != http.StatusOK {
		t.Fatalf("leave: %d", rec.Code)
	}
	left := request(t, h, http.MethodDelete, "/api/v1/communities/public-space/membership", nil, ownerCookie)
	if left.Code != http.StatusConflict {
		t.Fatalf("owner leave: %d %s", left.Code, left.Body.String())
	}
	var code struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(left.Body.Bytes(), &code); err != nil || code.Error.Code != "owner_cannot_leave" {
		t.Fatalf("owner leave error = %+v, %v", code, err)
	}

	// Moderators can be assigned only by the owner.
	if rec := request(t, h, http.MethodPost, "/api/v1/communities/public-space/moderators",
		map[string]any{"userId": member}, memberCookie); rec.Code != http.StatusForbidden {
		t.Fatalf("non-owner moderator assignment: %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, "/api/v1/communities/public-space/moderators",
		map[string]any{"userId": member}, ownerCookie); rec.Code != http.StatusOK {
		t.Fatalf("owner moderator assignment: %d", rec.Code)
	}
	if role := RoleFor(d, publicCommunityID, member); role != "moderator" {
		t.Fatalf("RoleFor = %q", role)
	}
}

func TestCommunityPostsPaginationAndShape(t *testing.T) {
	d, owner := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, owner)
	created := request(t, h, http.MethodPost, "/api/v1/communities", map[string]any{
		"name": "Posts Place", "slug": "posts-place", "visibility": "public",
	}, cookie)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	communityID := decodeBody[Community](t, created).ID

	for i := 0; i < 3; i++ {
		if _, err := d.DB.Exec(`
			INSERT INTO posts(author_id, community_id, type, title, body, created_at)
			VALUES(?, ?, 'text', ?, 'body', datetime('now'))`, owner, communityID, "post"); err != nil {
			t.Fatal(err)
		}
	}
	first := request(t, h, http.MethodGet, "/api/v1/communities/posts-place/posts?limit=2", nil, nil)
	if first.Code != http.StatusOK {
		t.Fatalf("community posts: %d %s", first.Code, first.Body.String())
	}
	page1 := decodeBody[page[Post]](t, first)
	if len(page1.Items) != 2 || page1.NextCursor == "" {
		t.Fatalf("page 1 = %+v", page1)
	}
	post := page1.Items[0]
	if post.VoteScore != 0 || post.MyVote != 0 || post.Media == nil || post.Anime == nil || post.Tags == nil || post.Poll != nil {
		t.Fatalf("post shape = %+v", post)
	}
	if post.Community == nil || post.Community.Slug != "posts-place" || post.Author.ID != owner {
		t.Fatalf("post relations = %+v", post)
	}
	second := request(t, h, http.MethodGet,
		"/api/v1/communities/posts-place/posts?limit=2&cursor="+page1.NextCursor, nil, nil)
	page2 := decodeBody[page[Post]](t, second)
	if len(page2.Items) != 1 || page2.NextCursor != "" {
		t.Fatalf("page 2 = %+v", page2)
	}
}
