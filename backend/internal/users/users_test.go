package users

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	migrations "github.com/Aswanidev-vs/promiseland/backend/migrations"
	"github.com/go-chi/chi/v5"
)

type testEnv struct {
	d      *shared.Deps
	server *httptest.Server
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	database, _ := testutil.TempDB(t, "users.db")
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
		RegisterProtected(r.With(shared.RequireAuth(deps)), deps)
	})
	server := httptest.NewServer(router)
	t.Cleanup(func() {
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

func seedUser(t *testing.T, env *testEnv, email, username, displayName, visibility string) int64 {
	t.Helper()
	result, err := env.d.DB.Exec(`
		INSERT INTO users (email, password_hash, username, display_name, email_verified, is_active)
		VALUES (?, 'not-a-real-hash', ?, ?, 1, 1)`, email, username, displayName)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	if _, err := env.d.DB.Exec(`INSERT INTO user_profiles (user_id, visibility) VALUES (?, ?)`, id, visibility); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	return id
}

func seedSession(t *testing.T, env *testEnv, userID int64) string {
	t.Helper()
	token := "session-token-" + hex.EncodeToString([]byte{byte(userID), byte(userID >> 8)}) + "-" + time.Now().Format("150405.000000000")
	sum := sha256.Sum256([]byte(token))
	if _, err := env.d.DB.Exec(`
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now','+1 day'))`, hex.EncodeToString(sum[:]), userID); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return token
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
		Details map[string]string `json:"details"`
	} `json:"error"`
}

func parseError(t *testing.T, body []byte) errorEnvelope {
	t.Helper()
	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error %q: %v", body, err)
	}
	return envelope
}

func TestProfileVisibilityAndShape(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	bob := seedUser(t, env, "bob@example.com", "bob", "Bob", "private")
	aliceToken := seedSession(t, env, alice)
	bobToken := seedSession(t, env, bob)

	status, body := env.do(t, "", http.MethodGet, "/api/v1/users/alice", nil)
	if status != http.StatusOK {
		t.Fatalf("public profile = %d (%s)", status, body)
	}
	profile := decodeJSON[UserProfile](t, body)
	if profile.Username != "alice" || profile.DisplayName != "Alice" {
		t.Errorf("profile = %+v", profile)
	}
	if profile.Interests == nil || profile.AnimeInterests == nil {
		t.Error("interests/anime_interests must serialise as [] not null")
	}
	if strings.Contains(string(body), `"email"`) {
		t.Errorf("public profile leaked email: %s", body)
	}
	for _, key := range []string{"follower_count", "following_count", "post_count", "is_following", "is_blocked"} {
		if !strings.Contains(string(body), `"`+key+`"`) {
			t.Errorf("profile missing %s: %s", key, body)
		}
	}

	status, body = env.do(t, "", http.MethodGet, "/api/v1/users/bob", nil)
	if status != http.StatusForbidden || parseError(t, body).Error.Code != codeProfilePrivate {
		t.Errorf("anonymous private profile = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodGet, "/api/v1/users/bob", nil)
	if status != http.StatusForbidden {
		t.Errorf("other user's private profile = %d %s", status, body)
	}
	status, body = env.do(t, bobToken, http.MethodGet, "/api/v1/users/bob", nil)
	if status != http.StatusOK {
		t.Fatalf("self private profile = %d (%s)", status, body)
	}
	self := decodeJSON[UserProfile](t, body)
	if self.Email != "bob@example.com" {
		t.Errorf("self profile should include email: %+v", self)
	}

	status, body = env.do(t, "", http.MethodGet, "/api/v1/users/nobody", nil)
	if status != http.StatusNotFound {
		t.Errorf("unknown user = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodGet, "/api/v1/users/me", nil)
	if status != http.StatusOK || decodeJSON[User](t, body).Username != "alice" {
		t.Errorf("me = %d %s", status, body)
	}
	// The static /users/me route must win over /users/{username} and stay
	// behind RequireAuth even for anonymous callers.
	status, body = env.do(t, "", http.MethodGet, "/api/v1/users/me", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("anonymous me = %d %s", status, body)
	}
}

func TestFollowUnfollowCounts(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	bob := seedUser(t, env, "bob@example.com", "bob", "Bob", "public")
	carol := seedUser(t, env, "carol@example.com", "carol", "Carol", "private")
	aliceToken := seedSession(t, env, alice)
	bobToken := seedSession(t, env, bob)

	status, body := env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(bob)+"/follow", nil)
	if status != http.StatusOK {
		t.Fatalf("follow = %d (%s)", status, body)
	}
	// Idempotent.
	if status, _ := env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(bob)+"/follow", nil); status != http.StatusOK {
		t.Errorf("repeat follow = %d", status)
	}
	status, body = env.do(t, aliceToken, http.MethodGet, "/api/v1/users/bob", nil)
	bobProfile := decodeJSON[UserProfile](t, body)
	if bobProfile.FollowerCount != 1 || !bobProfile.IsFollowing {
		t.Errorf("bob profile = %+v", bobProfile)
	}
	status, body = env.do(t, bobToken, http.MethodGet, "/api/v1/users/alice", nil)
	aliceProfile := decodeJSON[UserProfile](t, body)
	if aliceProfile.FollowingCount != 1 {
		t.Errorf("alice following count = %d", aliceProfile.FollowingCount)
	}

	var notifications int
	if err := env.d.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id = ? AND type = 'follow'`, bob).Scan(&notifications); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if notifications != 1 {
		t.Errorf("follow notifications = %d, want 1", notifications)
	}

	status, body = env.do(t, aliceToken, http.MethodDelete, "/api/v1/users/"+itoa(bob)+"/follow", nil)
	if status != http.StatusOK {
		t.Fatalf("unfollow = %d (%s)", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodGet, "/api/v1/users/bob", nil)
	profile := decodeJSON[UserProfile](t, body)
	if profile.FollowerCount != 0 || profile.IsFollowing {
		t.Errorf("profile after unfollow = %+v", profile)
	}

	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(alice)+"/follow", nil)
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != "cannot_follow_self" {
		t.Errorf("self follow = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(carol)+"/follow", nil)
	if status != http.StatusForbidden || parseError(t, body).Error.Code != codeProfilePrivate {
		t.Errorf("follow private = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/users/99999/follow", nil)
	if status != http.StatusNotFound {
		t.Errorf("follow unknown = %d %s", status, body)
	}
}

func TestBlockBehavior(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	bob := seedUser(t, env, "bob@example.com", "bob", "Bob", "public")
	aliceToken := seedSession(t, env, alice)
	bobToken := seedSession(t, env, bob)

	// Existing follows are removed by a block, in both directions.
	if status, _ := env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(bob)+"/follow", nil); status != http.StatusOK {
		t.Fatalf("initial follow = %d", status)
	}
	if status, _ := env.do(t, bobToken, http.MethodPost, "/api/v1/users/"+itoa(alice)+"/follow", nil); status != http.StatusOK {
		t.Fatalf("initial reverse follow = %d", status)
	}
	status, body := env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(bob)+"/block", nil)
	if status != http.StatusOK {
		t.Fatalf("block = %d (%s)", status, body)
	}
	var follows int
	if err := env.d.DB.QueryRow(`
		SELECT COUNT(*) FROM follows
		WHERE (follower_id = ? AND followed_id = ?) OR (follower_id = ? AND followed_id = ?)`,
		alice, bob, bob, alice).Scan(&follows); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if follows != 0 {
		t.Errorf("follows after block = %d, want 0", follows)
	}
	status, body = env.do(t, bobToken, http.MethodGet, "/api/v1/users/alice", nil)
	if status != http.StatusForbidden || parseError(t, body).Error.Code != codeBlocked {
		t.Errorf("blocked profile view = %d %s", status, body)
	}
	status, body = env.do(t, bobToken, http.MethodPost, "/api/v1/users/"+itoa(alice)+"/follow", nil)
	if status != http.StatusForbidden {
		t.Errorf("blocked follow = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(bob)+"/follow", nil)
	if status != http.StatusForbidden {
		t.Errorf("blocker follow = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodPost, "/api/v1/users/"+itoa(alice)+"/block", nil)
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != "cannot_block_self" {
		t.Errorf("self block = %d %s", status, body)
	}
	status, body = env.do(t, aliceToken, http.MethodDelete, "/api/v1/users/"+itoa(bob)+"/block", nil)
	if status != http.StatusOK {
		t.Fatalf("unblock = %d (%s)", status, body)
	}
	status, body = env.do(t, bobToken, http.MethodGet, "/api/v1/users/alice", nil)
	if status != http.StatusOK {
		t.Errorf("profile after unblock = %d %s", status, body)
	}
}

func TestPatchMe(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	token := seedSession(t, env, alice)

	status, body := env.do(t, token, http.MethodPatch, "/api/v1/users/me", map[string]any{
		"display_name": "Alice Updated",
		"bio":          "Cosplayer and illustrator.",
		"location":     "Leeds",
		"avatar_url":   "https://example.com/avatar.png",
		"banner_url":   "https://example.com/banner.png",
		"visibility":   "private",
		"discoverable": false,
	})
	if status != http.StatusOK {
		t.Fatalf("patch = %d (%s)", status, body)
	}
	updated := decodeJSON[User](t, body)
	if updated.DisplayName != "Alice Updated" || updated.Visibility != "private" || updated.Discoverable {
		t.Errorf("patched user = %+v", updated)
	}
	if updated.Bio == nil || *updated.Bio != "Cosplayer and illustrator." || updated.Location == nil || *updated.Location != "Leeds" {
		t.Errorf("patched profile = %+v", updated)
	}
	status, body = env.do(t, token, http.MethodGet, "/api/v1/users/alice", nil)
	if status != http.StatusOK {
		t.Fatalf("self private profile = %d (%s)", status, body)
	}

	for name, patch := range map[string]map[string]any{
		"visibility":   {"visibility": "secret"},
		"avatar":       {"avatar_url": "javascript:alert(1)"},
		"display name": {"display_name": "  "},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := env.do(t, token, http.MethodPatch, "/api/v1/users/me", patch)
			if status != http.StatusBadRequest || len(parseError(t, body).Error.Details) == 0 {
				t.Errorf("invalid patch = %d %s", status, body)
			}
		})
	}
}

func TestInterestsEndpoints(t *testing.T) {
	env := newTestEnv(t)
	alice := seedUser(t, env, "alice@example.com", "alice", "Alice", "public")
	token := seedSession(t, env, alice)

	status, body := env.do(t, token, http.MethodGet, "/api/v1/users/me/interests", nil)
	if status != http.StatusOK {
		t.Fatalf("list interests = %d (%s)", status, body)
	}
	interests := decodeJSON[[]Interest](t, body)
	if len(interests) < 14 {
		t.Errorf("interest count = %d, want >= 14", len(interests))
	}
	foundCosplay := false
	for _, interest := range interests {
		if interest.Name == "cosplay" && interest.Category != "" {
			foundCosplay = true
		}
	}
	if !foundCosplay {
		t.Error("seed interest 'cosplay' missing")
	}

	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/interests", []int64{1, 2, 2})
	if status != http.StatusOK {
		t.Fatalf("replace interests = %d (%s)", status, body)
	}
	profile := decodeJSON[UserProfile](t, body)
	if len(profile.Interests) != 2 {
		t.Errorf("interests after replace = %+v", profile.Interests)
	}

	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/interests", map[string]any{"ids": []int64{3}})
	if status != http.StatusOK || len(decodeJSON[UserProfile](t, body).Interests) != 1 {
		t.Errorf("wrapped interest payload = %d %s", status, body)
	}
	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/interests", []int64{99999})
	if status != http.StatusBadRequest {
		t.Errorf("unknown interest = %d %s", status, body)
	}

	var narutoID int64
	if err := env.d.DB.QueryRow(`SELECT id FROM anime_titles WHERE title = 'Naruto'`).Scan(&narutoID); err != nil {
		t.Fatalf("find seeded Naruto: %v", err)
	}
	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/anime-interests", []map[string]any{
		{"animeId": narutoID, "interestType": "Favorite"},
	})
	if status != http.StatusOK {
		t.Fatalf("replace anime interests = %d (%s)", status, body)
	}
	profile = decodeJSON[UserProfile](t, body)
	if len(profile.AnimeInterests) != 1 || profile.AnimeInterests[0].Anime.Title != "Naruto" ||
		profile.AnimeInterests[0].InterestType != "favorite" || profile.AnimeInterests[0].SortOrder != 0 {
		t.Errorf("anime interests = %+v", profile.AnimeInterests)
	}
	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/anime-interests", map[string]any{
		"items": []map[string]any{{"animeId": narutoID, "interestType": "watching"}},
	})
	if status != http.StatusOK || len(decodeJSON[UserProfile](t, body).AnimeInterests) != 1 {
		t.Errorf("wrapped anime interest payload = %d %s", status, body)
	}
	status, body = env.do(t, token, http.MethodPut, "/api/v1/users/me/anime-interests", []map[string]any{
		{"animeId": 99999, "interestType": "watching"},
	})
	if status != http.StatusBadRequest {
		t.Errorf("unknown anime = %d %s", status, body)
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}

func TestMigration001BackfillsLegacyUsernames(t *testing.T) {
	database, _ := testutil.TempDB(t, "legacy.db")
	foundation, err := migrations.FS.ReadFile("000_foundation.sql")
	if err != nil {
		t.Fatalf("read 000: %v", err)
	}
	if _, err := database.Exec(string(foundation)); err != nil {
		t.Fatalf("apply 000: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO users (email, password_hash, display_name)
		VALUES ('legacy@example.com', 'hash', 'Sakura Hana')`); err != nil {
		t.Fatalf("seed legacy user: %v", err)
	}
	content, err := migrations.FS.ReadFile("001_auth_users.sql")
	if err != nil {
		t.Fatalf("read 001: %v", err)
	}
	if _, err := database.Exec(string(content)); err != nil {
		t.Fatalf("apply 001: %v", err)
	}
	var username string
	if err := database.QueryRow(`SELECT username FROM users WHERE id = 1`).Scan(&username); err != nil {
		t.Fatalf("read backfilled username: %v", err)
	}
	if username != "sakura_hana1" {
		t.Errorf("backfilled username = %q, want %q", username, "sakura_hana1")
	}
	if _, err := database.Exec(`
		INSERT INTO users (email, password_hash, username, display_name)
		VALUES ('other@example.com', 'hash', 'sakura_hana1', 'Other')`); err == nil {
		t.Error("expected unique username violation")
	}
	if _, err := database.Exec(`INSERT INTO blocks (blocker_id, blocked_id) VALUES (1, 1)`); err == nil {
		t.Error("expected self-block check violation")
	}
	if _, err := database.Exec(`INSERT INTO interests (name, category) VALUES ('cosplay', 'creative')`); err == nil {
		t.Error("expected duplicate interest to be ignored")
	}
}
