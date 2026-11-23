package discovery

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

type testEnv struct {
	d      *shared.Deps
	server *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "discovery.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
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
		// Both entry points are mounted deliberately: handlers must stay
		// protected even when the orchestrator wires only Register.
		Register(r, deps)
		RegisterProtected(r.With(shared.RequireAuth(deps)), deps)
	})
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
		_ = database.Close()
	})
	return &testEnv{d: deps, server: server}
}

func (e *testEnv) do(t *testing.T, token, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.AddCookie(&http.Cookie{Name: shared.SessionCookieName, Value: token})
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, data
}

func (e *testEnv) exec(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := e.d.DB.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

func (e *testEnv) lastID(t *testing.T, query string, args ...any) int64 {
	t.Helper()
	result, err := e.d.DB.Exec(query, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func seedUser(t *testing.T, env *testEnv, username, visibility string, discoverable bool) int64 {
	t.Helper()
	flag := 0
	if discoverable {
		flag = 1
	}
	id := env.lastID(t, `
		INSERT INTO users (email, password_hash, username, display_name, email_verified, is_active, discoverable)
		VALUES (?, 'not-a-real-hash', ?, ?, 1, 1, ?)`,
		username+"@example.com", username, strings.ToUpper(username[:1])+username[1:], flag)
	env.exec(t, `INSERT INTO user_profiles (user_id, visibility, location) VALUES (?, ?, ?)`,
		id, visibility, "Secret Location "+username)
	return id
}

func seedSession(t *testing.T, env *testEnv, userID int64) string {
	t.Helper()
	token := fmt.Sprintf("session-%d-%d", userID, time.Now().UnixNano())
	sum := sha256.Sum256([]byte(token))
	env.exec(t, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now','+1 day'))`, hex.EncodeToString(sum[:]), userID)
	return token
}

func addAnime(t *testing.T, env *testEnv, title string) int64 {
	t.Helper()
	return env.lastID(t, `INSERT INTO anime_titles (title, format) VALUES (?, 'TV')`, title)
}

func addAnimeInterest(t *testing.T, env *testEnv, userID, animeID int64) {
	t.Helper()
	env.exec(t, `
		INSERT OR IGNORE INTO anime_interests (user_id, anime_title_id, interest_type, sort_order)
		VALUES (?, ?, 'watching', 0)`, userID, animeID)
}

func addInterest(t *testing.T, env *testEnv, name, category string) int64 {
	t.Helper()
	var id int64
	err := env.d.DB.QueryRow(`SELECT id FROM interests WHERE name = ? AND category = ?`, name, category).Scan(&id)
	if err == nil {
		return id
	}
	if err != sql.ErrNoRows {
		t.Fatalf("find interest: %v", err)
	}
	return env.lastID(t, `INSERT INTO interests (name, category) VALUES (?, ?)`, name, category)
}

func addUserInterest(t *testing.T, env *testEnv, userID, interestID int64) {
	t.Helper()
	env.exec(t, `INSERT OR IGNORE INTO user_interests (user_id, interest_id) VALUES (?, ?)`, userID, interestID)
}

func addCommunity(t *testing.T, env *testEnv, ownerID int64, name string) int64 {
	t.Helper()
	return env.lastID(t, `
		INSERT INTO communities (owner_id, name, slug, description)
		VALUES (?, ?, ?, '')`, ownerID, name, strings.ToLower(strings.ReplaceAll(name, " ", "-")))
}

func addMembership(t *testing.T, env *testEnv, userID, communityID int64) {
	t.Helper()
	env.exec(t, `
		INSERT OR REPLACE INTO community_memberships (community_id, user_id, role, status)
		VALUES (?, ?, 'member', 'active')`, communityID, userID)
}

func addCategoryPost(t *testing.T, env *testEnv, authorID int64, category string) {
	t.Helper()
	env.exec(t, `
		INSERT INTO posts (author_id, type, title, creator_category)
		VALUES (?, 'text', 'Creator post', ?)`, authorID, category)
}

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return value
}

type errorEnvelope struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Details map[string]string `json:"details"`
	} `json:"error"`
}

func parseError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	return decodeJSON[errorEnvelope](t, body)
}

func listFriends(t *testing.T, env *testEnv, token, query string) page[DiscoveryCandidate] {
	t.Helper()
	status, body := env.do(t, token, http.MethodGet, "/api/v1/discovery/friends"+query, nil)
	if status != http.StatusOK {
		t.Fatalf("friends = %d (%s)", status, body)
	}
	return decodeJSON[page[DiscoveryCandidate]](t, body)
}

func candidateIDs(items []DiscoveryCandidate) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.User.ID)
	}
	return ids
}

func containsID(ids []int64, id int64) bool {
	for _, value := range ids {
		if value == id {
			return true
		}
	}
	return false
}

func TestDiscoveryFriendsRankingExplanationAndOrdering(t *testing.T) {
	env := newTestEnv(t)
	viewer := seedUser(t, env, "viewer", "public", true)
	top := seedUser(t, env, "top", "public", true)
	medium := seedUser(t, env, "medium", "public", true)
	tied := seedUser(t, env, "tied", "public", true)
	empty := seedUser(t, env, "empty", "public", true)
	token := seedSession(t, env, viewer)

	sharedAnime := make([]int64, 8)
	for i := range sharedAnime {
		sharedAnime[i] = addAnime(t, env, fmt.Sprintf("Discovery Anime %d", i+1))
		addAnimeInterest(t, env, viewer, sharedAnime[i])
		addAnimeInterest(t, env, top, sharedAnime[i])
	}
	for i := 0; i < 3; i++ {
		addAnimeInterest(t, env, medium, sharedAnime[i])
	}
	// medium also shares one interest; tied shares three communities.
	interest := addInterest(t, env, "discovery-reading", "reading")
	addUserInterest(t, env, viewer, interest)
	addUserInterest(t, env, medium, interest)
	for i := 0; i < 3; i++ {
		community := addCommunity(t, env, viewer, fmt.Sprintf("Discovery Community %d", i+1))
		addMembership(t, env, viewer, community)
		addMembership(t, env, top, community)
		addMembership(t, env, tied, community)
	}

	friends := listFriends(t, env, token, "")
	ids := candidateIDs(friends.Items)
	want := []int64{top, medium, tied, empty}
	if fmt.Sprint(ids) != fmt.Sprint(want) {
		t.Fatalf("ordering = %v, want %v", ids, want)
	}
	if friends.Items[0].SharedCount != 11 {
		t.Errorf("top shared_count = %d, want 11", friends.Items[0].SharedCount)
	}
	if got, want := friends.Items[0].Explanation,
		"You both follow 8 of the same anime and 3 of the same communities"; got != want {
		t.Errorf("explanation = %q, want %q", got, want)
	}
	if len(friends.Items[0].Shared.Anime) != sharedNameCap {
		t.Errorf("shared anime names = %d, want %d", len(friends.Items[0].Shared.Anime), sharedNameCap)
	}
	if len(friends.Items[0].Shared.Communities) != 3 {
		t.Errorf("shared communities = %v", friends.Items[0].Shared.Communities)
	}
	if got, want := friends.Items[1].Explanation,
		"You both follow 3 of the same anime and 1 of the same interest"; got != want {
		t.Errorf("medium explanation = %q, want %q", got, want)
	}
	if friends.Items[3].SharedCount != 0 || friends.Items[3].Explanation != "" {
		t.Errorf("empty candidate = %+v", friends.Items[3])
	}
	if friends.NextCursor != "" {
		t.Errorf("next_cursor on final page = %q", friends.NextCursor)
	}

	// No location or email data is exposed anywhere in the payload.
	raw := string(mustBody(t, env, token, "/api/v1/discovery/friends"))
	if strings.Contains(raw, "Secret Location") || strings.Contains(raw, `"location"`) ||
		strings.Contains(raw, `"email"`) {
		t.Fatalf("discovery leaked private fields: %s", raw)
	}
}

func mustBody(t *testing.T, env *testEnv, token, path string) []byte {
	t.Helper()
	status, body := env.do(t, token, http.MethodGet, path, nil)
	if status != http.StatusOK {
		t.Fatalf("%s = %d (%s)", path, status, body)
	}
	return body
}

func TestDiscoveryFriendsExclusionsAndPendingStatus(t *testing.T) {
	env := newTestEnv(t)
	viewer := seedUser(t, env, "viewer", "public", true)
	private := seedUser(t, env, "private", "private", true)
	hidden := seedUser(t, env, "hidden", "public", false)
	blocked := seedUser(t, env, "blocked", "public", true)
	blockedMe := seedUser(t, env, "blockedme", "public", true)
	suppressed := seedUser(t, env, "suppressed", "public", true)
	accepted := seedUser(t, env, "accepted", "public", true)
	pendingIn := seedUser(t, env, "pendingin", "public", true)
	pendingOut := seedUser(t, env, "pendingout", "public", true)
	token := seedSession(t, env, viewer)

	env.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, viewer, blocked)
	env.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, blockedMe, viewer)
	env.exec(t, `INSERT INTO discovery_suppressions (viewer_id, candidate_id) VALUES (?, ?)`, viewer, suppressed)
	env.exec(t, `
		INSERT INTO connections (requester_id, addressee_id, status)
		VALUES (?, ?, 'accepted')`, viewer, accepted)
	env.exec(t, `
		INSERT INTO connections (requester_id, addressee_id, status)
		VALUES (?, ?, 'pending')`, pendingIn, viewer)
	env.exec(t, `
		INSERT INTO connections (requester_id, addressee_id, status)
		VALUES (?, ?, 'pending')`, viewer, pendingOut)

	ids := candidateIDs(listFriends(t, env, token, "").Items)
	for _, excluded := range []int64{viewer, private, hidden, blocked, blockedMe, suppressed, accepted} {
		if containsID(ids, excluded) {
			t.Errorf("candidate %d should be excluded; got %v", excluded, ids)
		}
	}
	statusByID := map[int64]string{}
	for _, candidate := range listFriends(t, env, token, "").Items {
		statusByID[candidate.User.ID] = candidate.ConnectionStatus
	}
	if statusByID[pendingIn] != "pending_in" || statusByID[pendingOut] != "pending_out" {
		t.Errorf("pending statuses = %v", statusByID)
	}

	// Unsuppressing is reversible and brings the candidate back.
	status, body := env.do(t, token, http.MethodDelete,
		fmt.Sprintf("/api/v1/discovery/%d/suppress", suppressed), nil)
	if status != http.StatusOK {
		t.Fatalf("unsuppress = %d (%s)", status, body)
	}
	if !containsID(candidateIDs(listFriends(t, env, token, "").Items), suppressed) {
		t.Error("unsuppressed candidate did not return")
	}

	// Self suppression and unknown targets are rejected.
	if status, body := env.do(t, token, http.MethodPost,
		fmt.Sprintf("/api/v1/discovery/%d/suppress", viewer), nil); status != http.StatusBadRequest {
		t.Errorf("self suppress = %d (%s)", status, body)
	}
	if status, body := env.do(t, token, http.MethodPost,
		"/api/v1/discovery/99999/suppress", nil); status != http.StatusNotFound {
		t.Errorf("missing target suppress = %d (%s)", status, body)
	}
}

func TestDiscoverySettings(t *testing.T) {
	env := newTestEnv(t)
	viewer := seedUser(t, env, "viewer", "public", true)
	token := seedSession(t, env, viewer)

	status, body := env.do(t, token, http.MethodGet, "/api/v1/discovery/settings", nil)
	if status != http.StatusOK || string(body) != "{\"discoverable\":true}\n" {
		t.Fatalf("initial settings = %d (%s)", status, body)
	}
	status, body = env.do(t, token, http.MethodPatch, "/api/v1/discovery/settings",
		map[string]any{"discoverable": false})
	if status != http.StatusOK || string(body) != "{\"discoverable\":false}\n" {
		t.Fatalf("patch settings = %d (%s)", status, body)
	}
	var value int
	if err := env.d.DB.QueryRow(`SELECT discoverable FROM users WHERE id = ?`, viewer).Scan(&value); err != nil || value != 0 {
		t.Fatalf("stored discoverable = %d, err %v", value, err)
	}
	for _, invalid := range []any{map[string]any{}, map[string]any{"discoverable": "yes"}} {
		status, body = env.do(t, token, http.MethodPatch, "/api/v1/discovery/settings", invalid)
		if status != http.StatusBadRequest {
			t.Errorf("invalid settings %v = %d (%s)", invalid, status, body)
			continue
		}
		if parseError(t, body).Error.Code != codeValidation {
			t.Errorf("invalid settings code = %s", parseError(t, body).Error.Code)
		}
	}
	if status, body := env.do(t, "", http.MethodGet, "/api/v1/discovery/settings", nil); status != http.StatusUnauthorized {
		t.Errorf("unauthenticated settings = %d (%s)", status, body)
	}
}
