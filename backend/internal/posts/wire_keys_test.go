package posts

import (
	"fmt"
	"net/http"
	"testing"
)

// The frontend client (frontend/src/api/posts.ts) posts snake_case keys. An
// unrecognised key is not an error: the post is created and the relation is
// silently dropped, which is how an `image` post ended up with no image.
// Assert on the payload the client actually sends, not the camelCase one the
// other tests use.
func TestPostCreateAcceptsFrontendSnakeCaseKeys(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)

	res, err := d.DB.Exec(`
		INSERT INTO media(owner_id, storage_key, mime_type, size_bytes, width, height, created_at)
		VALUES(?, 'pic.png', 'image/png', 100, 2, 2, datetime('now'))`, user)
	if err != nil {
		t.Fatal(err)
	}
	mediaID, _ := res.LastInsertId()

	if _, err := d.DB.Exec(`
		INSERT INTO communities(owner_id, name, slug, description) VALUES(?, 'Anime', 'anime', '')`,
		user); err != nil {
		t.Fatal(err)
	}
	animeRes, err := d.DB.Exec(`INSERT INTO anime_titles(title, format) VALUES('Barakamon', 'tv')`)
	if err != nil {
		t.Fatal(err)
	}
	animeID, _ := animeRes.LastInsertId()

	post := createTestPost(t, h, cookie, map[string]any{
		"community_slug":   "anime",
		"type":             "image",
		"title":            "barakamon",
		"body":             "mon",
		"link_url":         "https://example.test/barakamon",
		"media_ids":        []int64{mediaID},
		"anime_ids":        []int64{animeID},
		"tags":             []string{"anime"},
		"spoiler":          true,
		"spoiler_note":     "Ending",
		"creator_category": "fan_art",
	})

	if len(post.Media) != 1 || post.Media[0].URL != "/media/pic.png" {
		t.Fatalf("media_ids dropped: %+v", post.Media)
	}
	if post.Community == nil || post.Community.Slug != "anime" {
		t.Fatalf("community_slug dropped: %+v", post.Community)
	}
	if len(post.Anime) != 1 {
		t.Fatalf("anime_ids dropped: %+v", post.Anime)
	}
	if post.LinkURL != "https://example.test/barakamon" {
		t.Fatalf("link_url dropped: %q", post.LinkURL)
	}
	if post.SpoilerNote != "Ending" {
		t.Fatalf("spoiler_note dropped: %q", post.SpoilerNote)
	}
	if post.CreatorCategory != "fan_art" {
		t.Fatalf("creator_category dropped: %q", post.CreatorCategory)
	}
}

// PATCH shares the aliasing problem: updatePost(id, patch) sends Post-shaped
// snake_case keys, so a patch that clears media must still reach the handler.
func TestPostPatchAcceptsFrontendSnakeCaseKeys(t *testing.T) {
	d, user := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, user)

	res, err := d.DB.Exec(`
		INSERT INTO media(owner_id, storage_key, mime_type, size_bytes, width, height, created_at)
		VALUES(?, 'pic2.png', 'image/png', 100, 2, 2, datetime('now'))`, user)
	if err != nil {
		t.Fatal(err)
	}
	mediaID, _ := res.LastInsertId()

	post := createTestPost(t, h, cookie, map[string]any{
		"type": "text", "title": "before", "body": "body",
	})

	rec := request(t, h, http.MethodPatch, fmt.Sprintf("/api/v1/posts/%d", post.ID), map[string]any{
		"media_ids":    []int64{mediaID},
		"spoiler_note": "From the patch",
	}, cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}
	patched := decodeBody[Post](t, rec)
	if len(patched.Media) != 1 || patched.Media[0].URL != "/media/pic2.png" {
		t.Fatalf("media_ids dropped on patch: %+v", patched.Media)
	}
	if patched.SpoilerNote != "From the patch" {
		t.Fatalf("spoiler_note dropped on patch: %q", patched.SpoilerNote)
	}
}
