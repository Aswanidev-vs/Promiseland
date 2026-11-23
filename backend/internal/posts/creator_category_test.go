package posts

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func postBody(title, categoryKey, category string) map[string]any {
	return map[string]any{
		"type": "image", "title": title, "body": "creator work",
		categoryKey: category,
	}
}

func TestCreatePostCreatorCategoryAliases(t *testing.T) {
	d, userID := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, userID)

	cases := []struct {
		name       string
		key        string
		category   string
		wantStatus int
		wantValue  string
	}{
		{name: "camelCase", key: "creatorCategory", category: "cosplay", wantStatus: http.StatusCreated, wantValue: "cosplay"},
		{name: "snake_case", key: "creator_category", category: "fan_art", wantStatus: http.StatusCreated, wantValue: "fan_art"},
		{name: "omitted", key: "unused", category: "", wantStatus: http.StatusCreated, wantValue: ""},
		{name: "invalid", key: "creator_category", category: "nonsense", wantStatus: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := request(t, h, http.MethodPost, "/api/v1/posts", postBody(tc.name, tc.key, tc.category), cookie)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus != http.StatusCreated {
				return
			}
			post := decodeBody[Post](t, rec)
			if post.CreatorCategory != tc.wantValue {
				t.Errorf("creator_category = %q, want %q", post.CreatorCategory, tc.wantValue)
			}
			var raw struct {
				CreatorCategory string `json:"creator_category"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatal(err)
			}
			if raw.CreatorCategory != tc.wantValue {
				t.Errorf("wire creator_category = %q, want %q", raw.CreatorCategory, tc.wantValue)
			}
		})
	}
}

func TestListPostsCreatorCategoryFilterAliases(t *testing.T) {
	d, userID := testDeps(t)
	h := testRouter(d)
	cookie := authCookie(t, d, userID)

	cosplay := request(t, h, http.MethodPost, "/api/v1/posts", postBody("cosplay", "creator_category", "cosplay"), cookie)
	if cosplay.Code != http.StatusCreated {
		t.Fatalf("create cosplay: %s", cosplay.Body.String())
	}
	cosplayID := decodeBody[Post](t, cosplay).ID
	fanArt := request(t, h, http.MethodPost, "/api/v1/posts", postBody("fan art", "creatorCategory", "fan_art"), cookie)
	if fanArt.Code != http.StatusCreated {
		t.Fatalf("create fan art: %s", fanArt.Body.String())
	}

	for _, key := range []string{"creatorCategory", "creator_category"} {
		rec := request(t, h, http.MethodGet, "/api/v1/posts?"+key+"=cosplay", nil, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("filter %s: status %d", key, rec.Code)
		}
		page := decodeBody[page[Post]](t, rec)
		if len(page.Items) != 1 || page.Items[0].ID != cosplayID {
			t.Errorf("%s filter returned %+v", key, page.Items)
		}
	}
	invalid := request(t, h, http.MethodGet, "/api/v1/posts?creator_category=bogus", nil, cookie)
	if invalid.Code != http.StatusBadRequest {
		t.Errorf("invalid category status = %d", invalid.Code)
	}
	if !strings.Contains(invalid.Body.String(), "validation_error") {
		t.Errorf("invalid category body = %s", invalid.Body.String())
	}
}
