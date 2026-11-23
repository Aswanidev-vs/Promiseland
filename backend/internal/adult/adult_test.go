package adult

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
	"strings"
	"testing"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
	"github.com/Aswanidev-vs/promiseland/backend/internal/db"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/Aswanidev-vs/promiseland/backend/internal/testutil"
	"github.com/go-chi/chi/v5"
)

const testSecretLocation = "12 Hidden Lane, Secret City"

type testEnv struct {
	d      *shared.Deps
	server *httptest.Server
	bodies [][]byte
}

// newTestEnv applies ALL real migrations into a temporary SQLite database.
// assurance is the ADULT_AGE_ASSURANCE value; an empty value selects the
// deny-all default. protected mounts RegisterProtected instead of Register.
func newTestEnv(t *testing.T, cfg config.Config, assurance string, protected bool) *testEnv {
	t.Helper()
	t.Setenv(envAgeAssurance, assurance)

	database, _ := testutil.TempDB(t, "adult.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	deps := &shared.Deps{
		DB:     database,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: cfg,
	}
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		if protected {
			RegisterProtected(r.With(shared.RequireAuth(deps)), deps)
		} else {
			Register(r, deps)
		}
	})
	server := httptest.NewServer(router)
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})
	return &testEnv{d: deps, server: server}
}

func devConfig() config.Config {
	return config.Config{
		AppEnv:             "development",
		CORSOrigin:         "http://localhost:5173",
		FeatureAdultDating: true,
	}
}

func (e *testEnv) request(t *testing.T, token, method, path string, body any) (int, []byte) {
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
	e.bodies = append(e.bodies, data)
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

// assertNoSensitiveFields walks every JSON response produced by this env and
// rejects private-contact, location and fake-score fields (PRD §7.14, §12.4).
func (e *testEnv) assertNoSensitiveFields(t *testing.T) {
	t.Helper()
	forbidden := []string{
		`"location"`, `"latitude"`, `"longitude"`, `"lat"`, `"lng"`,
		`"phone"`, `"contact"`, `"email"`, `"birthday"`, `"date_of_birth"`,
		`"score"`, `"compatibility"`, `"percent"`, `"match_score"`,
		testSecretLocation,
	}
	for _, body := range e.bodies {
		lower := strings.ToLower(string(body))
		for _, key := range forbidden {
			if strings.Contains(lower, strings.ToLower(key)) {
				t.Fatalf("sensitive field %s leaked in response: %s", key, body)
			}
		}
	}
}

func decode[T any](t *testing.T, body []byte) T {
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

func seedUser(t *testing.T, env *testEnv, username string) int64 {
	t.Helper()
	env.exec(t, `
		INSERT INTO users (email, password_hash, username, display_name, email_verified, is_active)
		VALUES (?, 'not-a-real-hash', ?, ?, 1, 1)`,
		username+"@example.com", username, strings.ToUpper(username[:1])+username[1:])
	var id int64
	if err := env.d.DB.QueryRow(`SELECT id FROM users WHERE username = ?`, username).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	// A deliberately secret location proves adult responses never expose the
	// ordinary profile's location column.
	env.exec(t, `INSERT INTO user_profiles (user_id, visibility, location) VALUES (?, 'public', ?)`,
		id, testSecretLocation)
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

func putProfile(t *testing.T, env *testEnv, token, name, intent string) AdultProfile {
	t.Helper()
	status, body := env.request(t, token, http.MethodPut, "/api/v1/adult/profile", map[string]any{
		"displayName": name,
		"intent":      intent,
		"bio":         "I like anime and long walks.",
		"boundaries":  "No unexpected messages.",
	})
	if status != http.StatusOK {
		t.Fatalf("put profile = %d (%s)", status, body)
	}
	return decode[AdultProfile](t, body)
}

func like(t *testing.T, env *testEnv, token string, targetID int64, status string) (int, []byte) {
	t.Helper()
	return env.request(t, token, http.MethodPost,
		fmt.Sprintf("/api/v1/adult/like/%d", targetID), map[string]string{"status": status})
}

func listMatches(t *testing.T, env *testEnv, token string) page[AdultMatch] {
	t.Helper()
	status, body := env.request(t, token, http.MethodGet, "/api/v1/adult/matches", nil)
	if status != http.StatusOK {
		t.Fatalf("matches = %d (%s)", status, body)
	}
	return decode[page[AdultMatch]](t, body)
}

func expectError(t *testing.T, status int, body []byte, wantStatus int, wantCode string) {
	t.Helper()
	if status != wantStatus {
		t.Fatalf("status = %d, want %d (%s)", status, wantStatus, body)
	}
	env := decode[errorEnvelope](t, body)
	if env.Error.Code != wantCode {
		t.Fatalf("code = %q, want %q (%s)", env.Error.Code, wantCode, body)
	}
}

// ---------------------------------------------------------------------------
// 1. Feature gate: disabled by default, 404 for every /adult/* handler.
// ---------------------------------------------------------------------------

func TestFeatureFlagDisabledReturns404Everywhere(t *testing.T) {
	cfg := devConfig()
	cfg.FeatureAdultDating = false
	env := newTestEnv(t, cfg, "", false)

	cases := []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"status", http.MethodGet, "/api/v1/adult/status", nil},
		{"get profile", http.MethodGet, "/api/v1/adult/profile", nil},
		{"put profile", http.MethodPut, "/api/v1/adult/profile", map[string]string{"intent": "dating"}},
		{"like", http.MethodPost, "/api/v1/adult/like/1", map[string]string{"status": "like"}},
		{"matches", http.MethodGet, "/api/v1/adult/matches", nil},
		{"unmatch", http.MethodPost, "/api/v1/adult/matches/1/unmatch", nil},
		{"reports", http.MethodPost, "/api/v1/adult/reports", map[string]any{"targetId": 1, "reason": "spam"}},
		{"dev verify", http.MethodPost, "/api/v1/adult/dev/verify", nil},
		{"unknown", http.MethodGet, "/api/v1/adult/anything", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, body := env.request(t, "", tc.method, tc.path, tc.body)
			if status != http.StatusNotFound {
				t.Fatalf("status = %d, want 404 (%s)", status, body)
			}
			if tc.path == "/api/v1/adult/status" {
				got := decode[AdultStatus](t, body)
				if got.Enabled || got.Eligible || got.AgeAssurance != assuranceNone ||
					got.Message != "This feature is not available." {
					t.Fatalf("unexpected disabled status: %+v", got)
				}
				return
			}
			expectError(t, status, body, http.StatusNotFound, shared.CodeNotFound)
		})
	}
	env.assertNoSensitiveFields(t)
}

// The frontend consumes exact camelCase keys. Verify the raw JSON object, not
// just the decoded struct, for both the disabled and enabled responses.
func TestStatusJSONContractKeys(t *testing.T) {
	wantKeys := []string{"enabled", "ageAssurance", "eligible", "message"}

	check := func(t *testing.T, cfg config.Config, assurance string) {
		t.Helper()
		env := newTestEnv(t, cfg, assurance, false)
		status, body := env.request(t, "", http.MethodGet, "/api/v1/adult/status", nil)
		if status != http.StatusOK && status != http.StatusNotFound {
			t.Fatalf("status = %d (%s)", status, body)
		}
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			t.Fatalf("decode: %v (%s)", err, body)
		}
		for _, key := range wantKeys {
			if _, ok := raw[key]; !ok {
				t.Fatalf("missing key %q in %s", key, body)
			}
		}
		if _, ok := raw["age_assurance"]; ok {
			t.Fatalf("unexpected snake_case alias in %s", body)
		}
		env.assertNoSensitiveFields(t)
	}

	disabled := devConfig()
	disabled.FeatureAdultDating = false
	t.Run("disabled", func(t *testing.T) { check(t, disabled, "") })
	t.Run("enabled", func(t *testing.T) { check(t, devConfig(), "") })
	t.Run("enabled with stub", func(t *testing.T) { check(t, devConfig(), assuranceDevelopmentStub) })
}

// ---------------------------------------------------------------------------
// 2. Enabled but no provider: default implementation denies everyone.
// ---------------------------------------------------------------------------

func TestEnabledWithoutAssuranceDeniesEveryone(t *testing.T) {
	cfg := devConfig()
	env := newTestEnv(t, cfg, "", false)
	userID := seedUser(t, env, "alice")
	token := seedSession(t, env, userID)

	status, body := env.request(t, "", http.MethodGet, "/api/v1/adult/status", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d (%s)", status, body)
	}
	got := decode[AdultStatus](t, body)
	if !got.Enabled || got.AgeAssurance != assuranceNone || got.Eligible {
		t.Fatalf("unexpected status: %+v", got)
	}

	for _, tc := range []struct {
		name   string
		method string
		path   string
		body   any
	}{
		{"profile", http.MethodGet, "/api/v1/adult/profile", nil},
		{"like", http.MethodPost, "/api/v1/adult/like/2", map[string]string{"status": "like"}},
		{"matches", http.MethodGet, "/api/v1/adult/matches", nil},
		{"report", http.MethodPost, "/api/v1/adult/reports", map[string]any{"targetId": 2, "reason": "spam"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := env.request(t, token, tc.method, tc.path, tc.body)
			expectError(t, status, body, http.StatusForbidden, codeEligibility)
		})
	}

	// The dev endpoint exists in development but writing a verification row
	// cannot grant access while the stub is not selected.
	status, body = env.request(t, token, http.MethodPost, "/api/v1/adult/dev/verify", nil)
	if status != http.StatusOK {
		t.Fatalf("dev verify = %d (%s)", status, body)
	}
	status, body = env.request(t, token, http.MethodGet, "/api/v1/adult/profile", nil)
	expectError(t, status, body, http.StatusForbidden, codeEligibility)
	env.assertNoSensitiveFields(t)
}

// ---------------------------------------------------------------------------
// 3. Development stub happy path: verify -> opt in -> mutual like -> match ->
//    conversation -> unmatch -> report.
// ---------------------------------------------------------------------------

func TestDevelopmentStubFullFlow(t *testing.T) {
	env := newTestEnv(t, devConfig(), assuranceDevelopmentStub, false)

	alice := seedUser(t, env, "alice")
	bob := seedUser(t, env, "bob")
	carol := seedUser(t, env, "carol")
	aliceToken := seedSession(t, env, alice)
	bobToken := seedSession(t, env, bob)
	carolToken := seedSession(t, env, carol)

	// Status is safe and does not reveal eligibility to anonymous callers.
	status, body := env.request(t, "", http.MethodGet, "/api/v1/adult/status", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d (%s)", status, body)
	}
	anonStatus := decode[AdultStatus](t, body)
	if !anonStatus.Enabled || anonStatus.AgeAssurance != assuranceDevelopmentStub || anonStatus.Eligible {
		t.Fatalf("anonymous status: %+v", anonStatus)
	}

	// Not verified yet: no access, even though the stub is selected.
	status, body = env.request(t, aliceToken, http.MethodGet, "/api/v1/adult/profile", nil)
	expectError(t, status, body, http.StatusForbidden, codeEligibility)

	for _, token := range []string{aliceToken, bobToken, carolToken} {
		status, body = env.request(t, token, http.MethodPost, "/api/v1/adult/dev/verify", nil)
		if status != http.StatusOK {
			t.Fatalf("dev verify = %d (%s)", status, body)
		}
	}
	status, body = env.request(t, aliceToken, http.MethodGet, "/api/v1/adult/status", nil)
	verifiedStatus := decode[AdultStatus](t, body)
	if status != http.StatusOK || !verifiedStatus.Eligible {
		t.Fatalf("verified status = %d (%+v)", status, verifiedStatus)
	}

	// No adult profile exists yet; GET is 404 and nobody else can see one.
	status, body = env.request(t, aliceToken, http.MethodGet, "/api/v1/adult/profile", nil)
	expectError(t, status, body, http.StatusNotFound, shared.CodeNotFound)

	aliceProfile := putProfile(t, env, aliceToken, "Alice", "both")
	if aliceProfile.Visibility != "opt_in" || aliceProfile.Intent != "both" || aliceProfile.UserID != alice {
		t.Fatalf("profile = %+v", aliceProfile)
	}
	bobProfile := putProfile(t, env, bobToken, "Bob", "friendship")
	if bobProfile.UserID != bob || bobProfile.Visibility != "opt_in" {
		t.Fatalf("profile = %+v", bobProfile)
	}
	status, body = env.request(t, aliceToken, http.MethodGet, "/api/v1/adult/profile", nil)
	if status != http.StatusOK || decode[AdultProfile](t, body).UserID != alice {
		t.Fatalf("get own profile = %d (%s)", status, body)
	}

	// A pass never creates a match, even if the other side likes.
	status, body = like(t, env, aliceToken, bob, "pass")
	if status != http.StatusOK || decode[AdultLike](t, body).Status != "pass" {
		t.Fatalf("pass = %d (%s)", status, body)
	}
	status, body = like(t, env, bobToken, alice, "like")
	if status != http.StatusOK {
		t.Fatalf("like = %d (%s)", status, body)
	}
	matches := listMatches(t, env, aliceToken)
	if len(matches.Items) != 0 {
		t.Fatalf("pass created a match: %+v", matches.Items)
	}

	// Reciprocal like creates one match and one direct conversation.
	status, body = like(t, env, aliceToken, bob, "like")
	if status != http.StatusOK {
		t.Fatalf("mutual like = %d (%s)", status, body)
	}
	matches = listMatches(t, env, aliceToken)
	if len(matches.Items) != 1 {
		var matchCount, likeCount, convCount int
		_ = env.d.DB.QueryRow(`SELECT COUNT(*) FROM adult_matches`).Scan(&matchCount)
		_ = env.d.DB.QueryRow(`SELECT COUNT(*) FROM adult_likes`).Scan(&likeCount)
		_ = env.d.DB.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&convCount)
		t.Fatalf("matches = %+v (matches=%d likes=%d convs=%d)", matches.Items, matchCount, likeCount, convCount)
	}
	match := matches.Items[0]
	if match.ConversationID == nil || *match.ConversationID <= 0 {
		t.Fatalf("match has no conversation id: %+v", match)
	}
	if len(match.Users) != 2 || match.Users[0].ID != alice || match.Users[1].ID != bob {
		t.Fatalf("match users = %+v", match.Users)
	}
	if match.Users[0].AvatarURL == nil && match.Users[1].AvatarURL == nil {
		// Null avatar_url is valid; this only documents that it is the only
		// optional field on the wire.
		t.Log("both avatars null")
	}

	var conversationID, memberCount int64
	var convType, pair string
	if err := env.d.DB.QueryRow(`SELECT id, type, pair_key FROM conversations`).Scan(&conversationID, &convType, &pair); err != nil {
		t.Fatalf("conversation row: %v", err)
	}
	low, high := alice, bob
	if low > high {
		low, high = high, low
	}
	wantPair := fmt.Sprintf("%d:%d", low, high)
	if convType != "direct" || pair != wantPair || conversationID != *match.ConversationID {
		t.Fatalf("conversation = id %d type %q pair %q, want %d/direct/%q",
			conversationID, convType, pair, *match.ConversationID, wantPair)
	}
	if err := env.d.DB.QueryRow(`SELECT COUNT(*) FROM conversation_members WHERE conversation_id = ?`,
		conversationID).Scan(&memberCount); err != nil || memberCount != 2 {
		t.Fatalf("conversation members = %d (%v)", memberCount, err)
	}

	// Either party can unmatch; the thread id is reused on a fresh mutual match.
	status, body = env.request(t, bobToken, http.MethodPost,
		fmt.Sprintf("/api/v1/adult/matches/%d/unmatch", match.ID), nil)
	if status != http.StatusNoContent {
		t.Fatalf("unmatch = %d (%s)", status, body)
	}
	if matches := listMatches(t, env, aliceToken); len(matches.Items) != 0 {
		t.Fatalf("unmatched still listed: %+v", matches.Items)
	}
	var unmatched string
	if err := env.d.DB.QueryRow(`SELECT COALESCE(unmatched_at,'') FROM adult_matches WHERE id = ?`,
		match.ID).Scan(&unmatched); err != nil || unmatched == "" {
		t.Fatalf("unmatched_at = %q (%v)", unmatched, err)
	}
	var likes int
	if err := env.d.DB.QueryRow(`SELECT COUNT(*) FROM adult_likes`).Scan(&likes); err != nil || likes != 0 {
		t.Fatalf("likes after unmatch = %d (%v)", likes, err)
	}

	// A single fresh like does not immediately rematch.
	status, _ = like(t, env, aliceToken, bob, "like")
	if status != http.StatusOK {
		t.Fatalf("re-like = %d", status)
	}
	if matches := listMatches(t, env, aliceToken); len(matches.Items) != 0 {
		t.Fatalf("rematched after only one like")
	}
	status, _ = like(t, env, bobToken, alice, "like")
	if status != http.StatusOK {
		t.Fatalf("re-mutual like = %d", status)
	}
	matches = listMatches(t, env, aliceToken)
	if len(matches.Items) != 1 || matches.Items[0].ConversationID == nil ||
		*matches.Items[0].ConversationID != conversationID {
		t.Fatalf("rematch = %+v", matches.Items)
	}

	// High-priority adult safety report lands in the existing reports table.
	status, body = env.request(t, aliceToken, http.MethodPost, "/api/v1/adult/reports", map[string]any{
		"targetType": "adult_profile", "targetId": bob, "reason": "harassment", "details": "Please review.",
	})
	if status != http.StatusCreated {
		t.Fatalf("report = %d (%s)", status, body)
	}
	report := decode[reportResponse](t, body)
	if report.TargetType != "adult_profile" || report.Reason != "harassment" || report.Status != "open" {
		t.Fatalf("report = %+v", report)
	}
	if report.Details == nil {
		t.Fatal("report has no details")
	}
	var details map[string]string
	if err := json.Unmarshal([]byte(*report.Details), &details); err != nil {
		t.Fatalf("details JSON: %v (%q)", err, *report.Details)
	}
	if details["priority"] != "high" || details["context"] != "dating" {
		t.Fatalf("details = %+v", details)
	}
	var storedType string
	if err := env.d.DB.QueryRow(`SELECT target_type FROM reports WHERE id = ?`, report.ID).Scan(&storedType); err != nil {
		t.Fatalf("report row: %v", err)
	}
	if storedType != "adult_profile" {
		t.Fatalf("stored target_type = %q", storedType)
	}

	// Non-opted-in targets are invisible; self-likes are rejected.
	status, body = like(t, env, aliceToken, carol, "like")
	expectError(t, status, body, http.StatusNotFound, shared.CodeNotFound)

	status, body = like(t, env, aliceToken, alice, "like")
	expectError(t, status, body, http.StatusBadRequest, codeValidation)

	// Blocks hide a profile even when it has opted in.
	putProfile(t, env, carolToken, "Carol", "dating")
	env.exec(t, `INSERT INTO blocks (blocker_id, blocked_id) VALUES (?, ?)`, carol, alice)
	status, body = like(t, env, aliceToken, carol, "like")
	expectError(t, status, body, http.StatusNotFound, shared.CodeNotFound)
	env.exec(t, `DELETE FROM blocks WHERE blocker_id = ? AND blocked_id = ?`, carol, alice)
	status, body = like(t, env, aliceToken, carol, "like")
	if status != http.StatusOK {
		t.Fatalf("like after unblock = %d (%s)", status, body)
	}

	env.assertNoSensitiveFields(t)
}

// ---------------------------------------------------------------------------
// 4. Production: dev verify refuses, deny-all remains even if DB is tampered.
// ---------------------------------------------------------------------------

func TestProductionRefusesDevVerifyAndDeniesAll(t *testing.T) {
	cfg := devConfig()
	cfg.AppEnv = config.ProductionEnv
	env := newTestEnv(t, cfg, assuranceDevelopmentStub, false)
	userID := seedUser(t, env, "dave")
	token := seedSession(t, env, userID)

	status, body := env.request(t, token, http.MethodPost, "/api/v1/adult/dev/verify", nil)
	expectError(t, status, body, http.StatusNotFound, shared.CodeNotFound)

	status, body = env.request(t, "", http.MethodGet, "/api/v1/adult/status", nil)
	got := decode[AdultStatus](t, body)
	if status != http.StatusOK || got.AgeAssurance != assuranceNone || got.Enabled != true || got.Eligible {
		t.Fatalf("production status = %d (%+v)", status, got)
	}

	// Even a forged verification row cannot enable access without a provider.
	env.exec(t, `
		INSERT INTO adult_verifications (user_id, status, method, verified_at, expires_at)
		VALUES (?, 'verified', 'development_stub', datetime('now'), datetime('now','+1 year'))`, userID)
	status, body = env.request(t, token, http.MethodGet, "/api/v1/adult/profile", nil)
	expectError(t, status, body, http.StatusForbidden, codeEligibility)
	env.assertNoSensitiveFields(t)
}

// ---------------------------------------------------------------------------
// 5. Validation, safe schema and both registration entry points.
// ---------------------------------------------------------------------------

func TestValidationAndSchemaSafety(t *testing.T) {
	env := newTestEnv(t, devConfig(), assuranceDevelopmentStub, false)
	userID := seedUser(t, env, "erin")
	token := seedSession(t, env, userID)
	status, body := env.request(t, token, http.MethodPost, "/api/v1/adult/dev/verify", nil)
	if status != http.StatusOK {
		t.Fatalf("dev verify = %d (%s)", status, body)
	}

	for _, tc := range []struct {
		name string
		body map[string]any
	}{
		{"bad intent", map[string]any{"displayName": "Erin", "intent": "romance"}},
		{"long bio", map[string]any{"displayName": "Erin", "intent": "dating", "bio": strings.Repeat("x", 2001)}},
		{"long boundaries", map[string]any{"displayName": "Erin", "intent": "dating", "boundaries": strings.Repeat("y", 1001)}},
		{"empty name", map[string]any{"displayName": " ", "intent": "dating"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := env.request(t, token, http.MethodPut, "/api/v1/adult/profile", tc.body)
			expectError(t, status, body, http.StatusBadRequest, codeValidation)
		})
	}
	status, body = env.request(t, token, http.MethodPost, "/api/v1/adult/like/999", map[string]string{"status": "superlike"})
	expectError(t, status, body, http.StatusBadRequest, codeValidation)

	status, body = env.request(t, token, http.MethodPost, "/api/v1/adult/reports", map[string]any{
		"targetId": 999, "reason": "not_a_moderation_code",
	})
	expectError(t, status, body, http.StatusBadRequest, codeValidation)

	// Every adult table must be free of location/contact/birthday fields.
	forbiddenCols := []string{
		"location", "latitude", "longitude", "lat", "lng", "phone",
		"contact", "email", "birthday", "date_of_birth", "age",
	}
	for _, table := range []string{"adult_profiles", "adult_verifications", "adult_likes", "adult_matches"} {
		cols := tableColumns(t, env.d.DB, table)
		if len(cols) == 0 {
			t.Fatalf("table %s has no columns", table)
		}
		for _, col := range forbiddenCols {
			if cols[col] {
				t.Fatalf("forbidden column %s.%s exists", table, col)
			}
		}
	}
	env.assertNoSensitiveFields(t)
}

func tableColumns(t *testing.T, database *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := database.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		t.Fatalf("PRAGMA %s: %v", table, err)
	}
	defer func() { _ = rows.Close() }()
	result := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dflt any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan PRAGMA %s: %v", table, err)
		}
		result[strings.ToLower(name)] = true
	}
	return result
}

func TestRegisterProtectedEntryPoint(t *testing.T) {
	env := newTestEnv(t, devConfig(), assuranceDevelopmentStub, true)
	userID := seedUser(t, env, "frank")
	token := seedSession(t, env, userID)

	// The protected router wrapper itself requires a session; status returns
	// the normal enabled shape for an authenticated caller.
	status, body := env.request(t, token, http.MethodGet, "/api/v1/adult/status", nil)
	got := decode[AdultStatus](t, body)
	if status != http.StatusOK || !got.Enabled || got.AgeAssurance != assuranceDevelopmentStub {
		t.Fatalf("protected status = %d (%+v)", status, got)
	}
	status, body = env.request(t, token, http.MethodPost, "/api/v1/adult/dev/verify", nil)
	if status != http.StatusOK {
		t.Fatalf("protected dev verify = %d (%s)", status, body)
	}
	_ = putProfile(t, env, token, "Frank", "dating")
	env.assertNoSensitiveFields(t)
}
