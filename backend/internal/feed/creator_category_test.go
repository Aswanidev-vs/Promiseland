package feed

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape/testutil"
)

func TestFeedCreatorCategoryAliasesAndShape(t *testing.T) {
	d := testutil.Deps(t)
	aliceID, _ := testutil.User(t, d, "alice")

	cosplayID := insertPost(t, d, aliceID, nil, "cosplay work")
	testutil.Exec(t, d, `UPDATE posts SET creator_category = 'cosplay' WHERE id = ?`, cosplayID)
	_ = insertPost(t, d, aliceID, nil, "plain work")

	for _, key := range []string{"creatorCategory", "creator_category"} {
		status, page := getFeed(t, feedRouter(d), "/feed?limit=50&"+key+"=cosplay", "")
		if status != http.StatusOK {
			t.Fatalf("%s status = %d", key, status)
		}
		if len(page.Items) != 1 || page.Items[0].ID != cosplayID {
			t.Fatalf("%s filter returned %+v", key, page.Items)
		}
		if page.Items[0].CreatorCategory != "cosplay" {
			t.Errorf("creator_category = %q", page.Items[0].CreatorCategory)
		}
	}

	// The unfiltered feed must still contain both posts and expose the new key.
	status, page := getFeed(t, feedRouter(d), "/feed?limit=50", "")
	if status != http.StatusOK || len(page.Items) != 2 {
		t.Fatalf("unfiltered feed = %d, %+v", status, page.Items)
	}
	rec := httptest.NewRecorder()
	feedRouter(d).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/feed?limit=50", nil))
	var raw struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	for _, item := range raw.Items {
		if _, ok := item["creator_category"]; !ok {
			t.Errorf("feed post JSON missing creator_category: %v", item)
		}
	}
}
