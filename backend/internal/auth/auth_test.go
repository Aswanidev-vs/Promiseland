package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
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

type testEnv struct {
	d      *shared.Deps
	server *httptest.Server
	client *http.Client
}

func newTestEnv(t *testing.T, appEnv string) *testEnv {
	t.Helper()
	database, _ := testutil.TempDB(t, "auth.db")
	if err := db.Migrate(database, nil); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	deps := &shared.Deps{
		DB:  database,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Config: config.Config{
			AppEnv:     appEnv,
			CORSOrigin: "http://localhost:5173",
			// Tokens are only useful to the tests that assert on them; the
			// default deployment no longer returns them.
			DevExposeTokens: appEnv != config.ProductionEnv,
		},
	}
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) {
		Register(r, deps)
		RegisterProtected(r.With(shared.RequireAuth(deps)), deps)
	})
	server := httptest.NewServer(router)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	t.Cleanup(func() {
		server.Close()
	})
	return &testEnv{d: deps, server: server, client: client}
}

func (e *testEnv) do(t *testing.T, method, path string, body any) (int, http.Header, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
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
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, resp.Header, data
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
	var envelope errorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode error body %q: %v", body, err)
	}
	return envelope
}

func decodeJSON[T any](t *testing.T, body []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return value
}

func TestRegisterLoginMeLogout(t *testing.T) {
	env := newTestEnv(t, "development")
	credentials := map[string]any{
		"email":       "Sakura@Example.com",
		"password":    "correct horse battery",
		"username":    "Sakura_Art",
		"displayName": "Sakura the Artist",
	}

	status, headers, body := env.do(t, http.MethodPost, "/api/v1/auth/register", credentials)
	if status != http.StatusCreated {
		t.Fatalf("register status = %d (%s)", status, body)
	}
	registered := decodeJSON[registerResponse](t, body)
	if registered.Username != "sakura_art" || registered.Email != "sakura@example.com" {
		t.Errorf("register user = %+v", registered.User)
	}
	if registered.DisplayName != "Sakura the Artist" || registered.Visibility != "public" || !registered.Discoverable {
		t.Errorf("register user = %+v", registered.User)
	}
	if registered.AvatarURL != nil || registered.Bio != nil {
		t.Errorf("empty profile fields should be null: %+v", registered.User)
	}
	if registered.DevVerificationToken == "" {
		t.Error("development register should include devVerificationToken")
	}
	if !strings.Contains(string(body), `"avatar_url":null`) {
		t.Errorf("avatar_url should be explicit null: %s", body)
	}

	cookieHeader := headers.Get("Set-Cookie")
	for _, want := range []string{"pl_session=", "HttpOnly", "SameSite=Lax", "Path=/", "Max-Age=604800"} {
		if !strings.Contains(cookieHeader, want) {
			t.Errorf("cookie %q missing %q", cookieHeader, want)
		}
	}
	if strings.Contains(cookieHeader, "Secure") {
		t.Errorf("development cookie should not be Secure: %q", cookieHeader)
	}

	// The raw token is never stored; only its SHA-256 digest is.
	var storedHash string
	if err := env.d.DB.QueryRow(`SELECT token_hash FROM sessions LIMIT 1`).Scan(&storedHash); err != nil {
		t.Fatalf("read session hash: %v", err)
	}
	sum := sha256.Sum256([]byte(registered.DevVerificationToken))
	if storedHash == hex.EncodeToString(sum[:]) {
		t.Error("verification token was stored as a session")
	}
	if len(storedHash) != 64 {
		t.Errorf("session hash length = %d, want 64", len(storedHash))
	}

	status, _, body = env.do(t, http.MethodGet, "/api/v1/auth/me", nil)
	if status != http.StatusOK {
		t.Fatalf("me status = %d (%s)", status, body)
	}
	me := decodeJSON[User](t, body)
	if me.Email != "sakura@example.com" || me.Username != "sakura_art" {
		t.Errorf("me = %+v", me)
	}

	status, headers, _ = env.do(t, http.MethodPost, "/api/v1/auth/logout", nil)
	if status != http.StatusOK {
		t.Fatalf("logout status = %d", status)
	}
	if !strings.Contains(headers.Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("logout should clear the cookie: %q", headers.Get("Set-Cookie"))
	}
	status, _, _ = env.do(t, http.MethodGet, "/api/v1/auth/me", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("me after logout = %d, want 401", status)
	}

	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"email": credentials["email"], "password": credentials["password"],
	})
	if status != http.StatusOK {
		t.Fatalf("login status = %d (%s)", status, body)
	}
	loggedIn := decodeJSON[User](t, body)
	if loggedIn.Email != "sakura@example.com" {
		t.Errorf("login user = %+v", loggedIn)
	}
	status, _, _ = env.do(t, http.MethodGet, "/api/v1/auth/me", nil)
	if status != http.StatusOK {
		t.Errorf("me after login = %d, want 200", status)
	}
}

func TestRegisterValidationAndDuplicates(t *testing.T) {
	env := newTestEnv(t, "development")
	base := map[string]any{
		"email": "user@example.com", "password": "password123",
		"username": "user_one", "displayName": "User One",
	}

	tests := []struct {
		name  string
		patch map[string]any
		field string
	}{
		{name: "bad email", patch: map[string]any{"email": "not-an-email"}, field: "email"},
		{name: "short password", patch: map[string]any{"password": "short"}, field: "password"},
		{name: "bad username", patch: map[string]any{"username": "Bad Name!"}, field: "username"},
		{name: "short username", patch: map[string]any{"username": "ab"}, field: "username"},
		{name: "empty display name", patch: map[string]any{"displayName": "   "}, field: "displayName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make(map[string]any, len(base))
			for k, v := range base {
				payload[k] = v
			}
			for k, v := range tt.patch {
				payload[k] = v
			}
			status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", payload)
			if status != http.StatusBadRequest {
				t.Fatalf("status = %d (%s)", status, body)
			}
			envelope := parseError(t, body)
			if envelope.Error.Code != codeValidation || envelope.Error.Details[tt.field] == "" {
				t.Errorf("error = %+v, want field %s", envelope.Error, tt.field)
			}
		})
	}

	// A fresh limiter: the five invalid attempts above used the register quota.
	dupEnv := newTestEnv(t, "development")
	if status, _, body := dupEnv.do(t, http.MethodPost, "/api/v1/auth/register", base); status != http.StatusCreated {
		t.Fatalf("initial register = %d (%s)", status, body)
	}
	duplicateEmail := map[string]any{
		"email": "USER@example.com", "password": "password123",
		"username": "user_two", "displayName": "User Two",
	}
	status, _, body := dupEnv.do(t, http.MethodPost, "/api/v1/auth/register", duplicateEmail)
	if status != http.StatusConflict || parseError(t, body).Error.Code != codeEmailTaken {
		t.Errorf("duplicate email = %d %s", status, body)
	}
	duplicateUsername := map[string]any{
		"email": "other@example.com", "password": "password123",
		"username": "USER_ONE", "displayName": "Other",
	}
	status, _, body = dupEnv.do(t, http.MethodPost, "/api/v1/auth/register", duplicateUsername)
	if status != http.StatusConflict || parseError(t, body).Error.Code != codeUsernameTaken {
		t.Errorf("duplicate username = %d %s", status, body)
	}
}

func TestLoginDoesNotRevealRegisteredEmail(t *testing.T) {
	env := newTestEnv(t, "development")
	if status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "known@example.com", "password": "password123",
		"username": "known_user", "displayName": "Known",
	}); status != http.StatusCreated {
		t.Fatalf("register = %d (%s)", status, body)
	}
	for name, payload := range map[string]map[string]any{
		"unknown email":  {"email": "unknown@example.com", "password": "password123"},
		"wrong password": {"email": "known@example.com", "password": "wrong-password"},
	} {
		t.Run(name, func(t *testing.T) {
			status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/login", payload)
			if status != http.StatusUnauthorized || parseError(t, body).Error.Code != codeInvalidCredentials {
				t.Errorf("login = %d %s", status, body)
			}
			if strings.Contains(strings.ToLower(string(body)), "registered") {
				t.Errorf("login response leaks registration state: %s", body)
			}
		})
	}
}

func TestVerifyEmailIsSingleUse(t *testing.T) {
	env := newTestEnv(t, "development")
	status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "verify@example.com", "password": "password123",
		"username": "verify_user", "displayName": "Verify",
	})
	if status != http.StatusCreated {
		t.Fatalf("register = %d (%s)", status, body)
	}
	token := decodeJSON[registerResponse](t, body).DevVerificationToken

	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/verify", map[string]any{"token": token})
	if status != http.StatusOK {
		t.Fatalf("verify = %d (%s)", status, body)
	}
	var verified int
	if err := env.d.DB.QueryRow(`SELECT email_verified FROM users WHERE username = 'verify_user'`).Scan(&verified); err != nil {
		t.Fatalf("read email_verified: %v", err)
	}
	if verified != 1 {
		t.Errorf("email_verified = %d, want 1", verified)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/verify", map[string]any{"token": token})
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeInvalidToken {
		t.Errorf("reused token = %d %s", status, body)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/verify", map[string]any{"token": "not-a-token"})
	if status != http.StatusBadRequest {
		t.Errorf("bad token = %d %s", status, body)
	}

	// An expired, unused token must also be rejected.
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "expire@example.com", "password": "password123",
		"username": "expire_user", "displayName": "Expire",
	})
	if status != http.StatusCreated {
		t.Fatalf("second register = %d (%s)", status, body)
	}
	expiredToken := decodeJSON[registerResponse](t, body).DevVerificationToken
	sum := sha256.Sum256([]byte(expiredToken))
	if _, err := env.d.DB.Exec(`UPDATE email_verification_tokens SET expires_at = datetime('now','-1 minute') WHERE token_hash = ?`,
		hex.EncodeToString(sum[:])); err != nil {
		t.Fatalf("expire verification token: %v", err)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/verify", map[string]any{"token": expiredToken})
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeInvalidToken {
		t.Errorf("expired verification token = %d %s", status, body)
	}
}

func TestPasswordResetSingleUseAndExpiry(t *testing.T) {
	env := newTestEnv(t, "development")
	if status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "reset@example.com", "password": "old-password",
		"username": "reset_user", "displayName": "Reset",
	}); status != http.StatusCreated {
		t.Fatalf("register = %d (%s)", status, body)
	}

	status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/password-reset/request",
		map[string]any{"email": "nobody@example.com"})
	if status != http.StatusAccepted {
		t.Fatalf("unknown email request = %d (%s)", status, body)
	}
	if strings.Contains(string(body), "devResetToken") {
		t.Errorf("unknown email should not return a token: %s", body)
	}

	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/request",
		map[string]any{"email": "reset@example.com"})
	if status != http.StatusAccepted {
		t.Fatalf("known email request = %d (%s)", status, body)
	}
	token := decodeJSON[passwordResetResponse](t, body).DevResetToken
	if token == "" {
		t.Fatal("missing devResetToken")
	}

	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/confirm",
		map[string]any{"token": token, "password": "short"})
	if status != http.StatusBadRequest || parseError(t, body).Error.Details["password"] == "" {
		t.Errorf("weak password = %d %s", status, body)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/confirm",
		map[string]any{"token": token, "password": "new-password"})
	if status != http.StatusOK {
		t.Fatalf("confirm = %d (%s)", status, body)
	}
	// The reset invalidated the session created at registration.
	status, _, _ = env.do(t, http.MethodGet, "/api/v1/auth/me", nil)
	if status != http.StatusUnauthorized {
		t.Errorf("me after password reset = %d, want 401", status)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/confirm",
		map[string]any{"token": token, "password": "another-password"})
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeInvalidToken {
		t.Errorf("reused reset token = %d %s", status, body)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]any{"email": "reset@example.com", "password": "old-password"})
	if status != http.StatusUnauthorized {
		t.Errorf("old password login = %d %s", status, body)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/login",
		map[string]any{"email": "reset@example.com", "password": "new-password"})
	if status != http.StatusOK {
		t.Errorf("new password login = %d %s", status, body)
	}

	// Expired token.
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/request",
		map[string]any{"email": "reset@example.com"})
	if status != http.StatusAccepted {
		t.Fatalf("second reset request = %d (%s)", status, body)
	}
	expiredToken := decodeJSON[passwordResetResponse](t, body).DevResetToken
	if _, err := env.d.DB.Exec(`UPDATE password_reset_tokens SET expires_at = datetime('now','-1 minute')`); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/confirm",
		map[string]any{"token": expiredToken, "password": "yet-another-password"})
	if status != http.StatusBadRequest || parseError(t, body).Error.Code != codeInvalidToken {
		t.Errorf("expired reset token = %d %s", status, body)
	}
}

func TestRateLimits(t *testing.T) {
	env := newTestEnv(t, "development")
	var lastCode string
	for i := 0; i < 6; i++ {
		status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
			"email":       "rate" + string(rune('a'+i)) + "@example.com",
			"password":    "password123",
			"username":    "rate_user_" + string(rune('a'+i)),
			"displayName": "Rate User",
		})
		if i == 5 {
			if status != http.StatusTooManyRequests {
				t.Fatalf("6th register = %d (%s)", status, body)
			}
			lastCode = parseError(t, body).Error.Code
		} else if status != http.StatusCreated {
			t.Fatalf("register %d = %d (%s)", i+1, status, body)
		}
	}
	if lastCode != codeRateLimited {
		t.Errorf("rate limit code = %q", lastCode)
	}

	for i := 0; i < 11; i++ {
		status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/login", map[string]any{
			"email": "nobody@example.com", "password": "password123",
		})
		if i == 10 {
			if status != http.StatusTooManyRequests || parseError(t, body).Error.Code != codeRateLimited {
				t.Errorf("11th login = %d %s", status, body)
			}
			return
		}
		if status != http.StatusUnauthorized {
			t.Fatalf("login %d = %d (%s)", i+1, status, body)
		}
	}
}

func TestProductionNeverReturnsDevTokens(t *testing.T) {
	env := newTestEnv(t, "production")
	status, headers, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "prod@example.com", "password": "password123",
		"username": "prod_user", "displayName": "Prod",
	})
	if status != http.StatusCreated {
		t.Fatalf("register = %d (%s)", status, body)
	}
	if strings.Contains(string(body), "devVerificationToken") {
		t.Errorf("production register leaked dev token: %s", body)
	}
	if !strings.Contains(headers.Get("Set-Cookie"), "Secure") {
		t.Errorf("production cookie must be Secure: %q", headers.Get("Set-Cookie"))
	}
	status, _, body = env.do(t, http.MethodPost, "/api/v1/auth/password-reset/request",
		map[string]any{"email": "prod@example.com"})
	if status != http.StatusAccepted || strings.Contains(string(body), "devResetToken") {
		t.Errorf("production reset leaked token: %d %s", status, body)
	}
}

// The dev-token convenience must be requested explicitly: a deployment that
// only forgot APP_ENV used to hand reset tokens to anonymous callers.
func TestDevTokensStayOffUnlessEnabled(t *testing.T) {
	env := newTestEnv(t, "development")
	env.d.Config.DevExposeTokens = false

	if status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/register", map[string]any{
		"email": "no-token@example.com", "password": "correct horse battery",
		"username": "no_token", "displayName": "No Token",
	}); status != http.StatusCreated || strings.Contains(string(body), "devVerificationToken") {
		t.Fatalf("register leaked a dev token: %d %s", status, body)
	}

	if status, _, body := env.do(t, http.MethodPost, "/api/v1/auth/password-reset/request",
		map[string]any{"email": "no-token@example.com"}); status != http.StatusAccepted || strings.Contains(string(body), "devResetToken") {
		t.Fatalf("reset request leaked a dev token: %d %s", status, body)
	}
}
