package anime

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	database, _ := testutil.TempDB(t, "anime.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	deps := &shared.Deps{
		DB:     database,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: config.Config{AppEnv: "development", CORSOrigin: "http://localhost:5173"},
	}
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		Register(r, deps)
		RegisterProtected(r, deps)
	})
	return router
}

func get(t *testing.T, handler http.Handler, path string) (int, []byte) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return rec.Code, body
}

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return value
}

func TestSearchIsCaseInsensitiveAndEscapesWildcards(t *testing.T) {
	handler := newTestHandler(t)

	status, body := get(t, handler, "/api/v1/anime?q=naruto")
	if status != http.StatusOK {
		t.Fatalf("search status = %d (%s)", status, body)
	}
	page := decodeJSON[Page[Anime]](t, body)
	if len(page.Items) != 2 || page.Items[0].Title != "Naruto" || page.Items[1].Title != "Naruto: Shippuden" {
		t.Errorf("naruto results = %+v", page.Items)
	}
	if page.NextCursor != "" {
		t.Errorf("next_cursor = %q, want empty", page.NextCursor)
	}

	status, body = get(t, handler, "/api/v1/anime?q=SHINGEKI")
	if status != http.StatusOK {
		t.Fatalf("alternate title search = %d (%s)", status, body)
	}
	alternate := decodeJSON[Page[Anime]](t, body)
	if len(alternate.Items) != 1 || alternate.Items[0].Title != "Attack on Titan" {
		t.Errorf("alternate title search = %+v", alternate.Items)
	}

	// A literal % or _ must not be treated as a wildcard. No seeded title
	// contains either character, so both searches must return nothing.
	for _, query := range []string{"%25", "_"} {
		status, body = get(t, handler, "/api/v1/anime?q="+query)
		if status != http.StatusOK {
			t.Fatalf("wildcard search %q = %d (%s)", query, status, body)
		}
		results := decodeJSON[Page[Anime]](t, body)
		if len(results.Items) != 0 {
			t.Errorf("query %q unexpectedly matched %+v", query, results.Items)
		}
	}

	status, body = get(t, handler, "/api/v1/anime")
	if status != http.StatusOK {
		t.Fatalf("empty query = %d (%s)", status, body)
	}
	all := decodeJSON[Page[Anime]](t, body)
	if len(all.Items) != 18 {
		t.Errorf("seeded title count = %d, want 18", len(all.Items))
	}
	if !strings.Contains(string(body), `"next_cursor":""`) {
		t.Errorf("next_cursor must always be present: %s", body)
	}
}

func TestSearchPagination(t *testing.T) {
	handler := newTestHandler(t)
	seen := map[int64]bool{}
	cursor := ""
	pages := 0
	for {
		path := "/api/v1/anime?limit=5"
		if cursor != "" {
			path += "&cursor=" + cursor
		}
		status, body := get(t, handler, path)
		if status != http.StatusOK {
			t.Fatalf("page %d status = %d (%s)", pages+1, status, body)
		}
		page := decodeJSON[Page[Anime]](t, body)
		if len(page.Items) > 5 {
			t.Fatalf("page size = %d, want <= 5", len(page.Items))
		}
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Errorf("duplicate id %d across pages", item.ID)
			}
			seen[item.ID] = true
		}
		pages++
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
		if pages > 10 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 18 {
		t.Errorf("total results across pages = %d, want 18", len(seen))
	}

	status, body := get(t, handler, "/api/v1/anime?cursor=!!!not-base64")
	if status != http.StatusBadRequest {
		t.Errorf("invalid cursor = %d %s", status, body)
	}
}

func TestGetAnimeAndGenres(t *testing.T) {
	handler := newTestHandler(t)

	status, body := get(t, handler, "/api/v1/anime/not-a-number")
	if status != http.StatusBadRequest {
		t.Errorf("invalid id = %d %s", status, body)
	}
	status, body = get(t, handler, "/api/v1/anime/99999")
	if status != http.StatusNotFound {
		t.Errorf("unknown id = %d %s", status, body)
	}

	// Discover a valid id through search rather than assuming seed ordering.
	status, body = get(t, handler, "/api/v1/anime?q=Spirited%20Away")
	if status != http.StatusOK {
		t.Fatalf("find title = %d (%s)", status, body)
	}
	found := decodeJSON[Page[Anime]](t, body)
	if len(found.Items) != 1 {
		t.Fatalf("find title = %+v", found.Items)
	}
	id := found.Items[0].ID
	status, body = get(t, handler, "/api/v1/anime/"+strconv.FormatInt(id, 10))
	if status != http.StatusOK {
		t.Fatalf("get by id = %d (%s)", status, body)
	}
	anime := decodeJSON[Anime](t, body)
	if anime.Title != "Spirited Away" || anime.Format != "Film" || anime.ExternalID != "" {
		t.Errorf("anime = %+v", anime)
	}
	for _, key := range []string{"external_id", "alternate_title", "format"} {
		if !strings.Contains(string(body), `"`+key+`"`) {
			t.Errorf("anime missing %s: %s", key, body)
		}
	}

	status, body = get(t, handler, "/api/v1/genres")
	if status != http.StatusOK {
		t.Fatalf("genres = %d (%s)", status, body)
	}
	genres := decodeJSON[[]Genre](t, body)
	if len(genres) < 14 {
		t.Errorf("genre count = %d, want >= 14", len(genres))
	}
}
