package communities

import (
	"net/http"
	"testing"
)

// The frontend client sends snake_case request keys (src/api/communities.ts).
// An unrecognised key is not an error, so the value is dropped: the posting
// toggle silently did nothing and adding a moderator always failed validation.
func TestCommunityFrontendSnakeCaseKeys(t *testing.T) {
	d, owner := testDeps(t)
	h := testRouter(d)
	ownerCookie := authCookie(t, d, owner)

	rec := request(t, h, http.MethodPost, "/api/v1/communities", map[string]any{
		"name": "Anime", "slug": "anime", "description": "", "visibility": "public",
	}, ownerCookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create community: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodPatch, "/api/v1/communities/anime", map[string]any{
		"posting_enabled": false,
	}, ownerCookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch posting_enabled: %d %s", rec.Code, rec.Body.String())
	}
	// The Community response shape has no posting_enabled field, so assert the
	// stored value: that is what the write path is responsible for.
	var postingEnabled int
	if err := d.DB.QueryRow(`SELECT posting_enabled FROM communities WHERE slug = 'anime'`).
		Scan(&postingEnabled); err != nil {
		t.Fatal(err)
	}
	if postingEnabled != 0 {
		t.Fatal("posting_enabled: false was dropped, posting is still enabled")
	}

	member := insertUser(t, d, "mod")
	rec = request(t, h, http.MethodPost, "/api/v1/communities/anime/moderators",
		map[string]any{"user_id": member}, ownerCookie)
	if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
		t.Fatalf("add moderator by user_id: %d %s", rec.Code, rec.Body.String())
	}
	var role string
	if err := d.DB.QueryRow(`
		SELECT role FROM community_memberships WHERE community_id = 1 AND user_id = ?`,
		member).Scan(&role); err != nil {
		t.Fatalf("membership row missing: %v", err)
	}
	if role != "moderator" {
		t.Fatalf("role = %q", role)
	}
}
