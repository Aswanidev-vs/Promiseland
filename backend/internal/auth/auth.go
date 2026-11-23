// Package auth implements registration, sessions, email verification and
// password reset for the R1 social foundation.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	codeValidation         = "validation_error"
	codeInvalidCredentials = "invalid_credentials"
	codeUsernameTaken      = "username_taken"
	codeEmailTaken         = "email_taken"
	codeInvalidToken       = "invalid_token"
	codeRateLimited        = "rate_limited"
)

const userColumns = `
	SELECT u.id, u.email, u.username, u.display_name,
	       p.avatar_url, p.banner_url, p.bio, p.location,
	       COALESCE(p.visibility, 'public'), u.discoverable, u.created_at
	FROM users u
	LEFT JOIN user_profiles p ON p.user_id = u.id`

// dummyHash is compared against on failed lookups so unknown-email and
// wrong-password requests take a similar amount of time. It is a throwaway
// bcrypt hash of a string nobody can use to sign in.
var dummyHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("promisedland-dummy-password"), bcrypt.MinCost)
	if err != nil {
		return ""
	}
	return string(hash)
}()

type service struct {
	d       *shared.Deps
	limiter *rateLimiter
}

// Register mounts public authentication routes under /auth.
func Register(r chi.Router, d *shared.Deps) {
	s := &service{d: d, limiter: newRateLimiter()}
	r.With(s.limiter.middleware("register", 5)).Post("/auth/register", s.register)
	r.With(s.limiter.middleware("login", 10)).Post("/auth/login", s.login)
	r.Post("/auth/verify", s.verify)
	r.With(s.limiter.middleware("reset-request", 5)).Post("/auth/password-reset/request", s.passwordResetRequest)
	r.With(s.limiter.middleware("reset-confirm", 5)).Post("/auth/password-reset/confirm", s.passwordResetConfirm)
}

// RegisterProtected mounts routes requiring a valid session. The orchestrator
// passes a router already wrapped with shared.RequireAuth.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	s := &service{d: d, limiter: newRateLimiter()}
	r.Post("/auth/logout", s.logout)
	r.Get("/auth/me", s.me)
}

func writeValidation(w http.ResponseWriter, details validationErrors) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
		"Please correct the highlighted fields.", map[string]string(details))
}

func internalError(w http.ResponseWriter, d *shared.Deps, action string, err error) {
	if d != nil && d.Log != nil {
		d.Log.Error("auth: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

func (s *service) userByID(ctx context.Context, id int64) (User, error) {
	var user User
	var avatar, banner, bio, location sql.NullString
	err := s.d.DB.QueryRowContext(ctx, userColumns+` WHERE u.id = ?`, id).Scan(
		&user.ID, &user.Email, &user.Username, &user.DisplayName,
		&avatar, &banner, &bio, &location,
		&user.Visibility, &user.Discoverable, &user.CreatedAt,
	)
	if err != nil {
		return User{}, err
	}
	user.AvatarURL = stringPtr(avatar)
	user.BannerURL = stringPtr(banner)
	user.Bio = stringPtr(bio)
	user.Location = stringPtr(location)
	return user, nil
}

func conflictCode(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "users.email"):
		return codeEmailTaken
	case strings.Contains(message, "users.username"):
		return codeUsernameTaken
	default:
		return ""
	}
}

func (s *service) register(w http.ResponseWriter, r *http.Request) {
	var in registerRequest
	if !shared.DecodeJSON(w, r, &in) {
		return
	}

	details := make(validationErrors)
	email := validateEmail(in.Email, details)
	password := in.Password // never trim or log credentials
	username := validateUsername(in.Username, details)
	displayName := validateDisplayName(in.DisplayName, details)
	validatePassword(password, details)
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	var exists int
	err := s.d.DB.QueryRowContext(r.Context(),
		`SELECT 1 FROM users WHERE email = ? LIMIT 1`, email).Scan(&exists)
	if err == nil {
		shared.WriteError(w, http.StatusConflict, codeEmailTaken, "An account with that email already exists.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		internalError(w, s.d, "register: check email", err)
		return
	}
	err = s.d.DB.QueryRowContext(r.Context(),
		`SELECT 1 FROM users WHERE username = ? LIMIT 1`, username).Scan(&exists)
	if err == nil {
		shared.WriteError(w, http.StatusConflict, codeUsernameTaken, "That username is already taken.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		internalError(w, s.d, "register: check username", err)
		return
	}

	hash, err := shared.HashPassword(password)
	if err != nil {
		internalError(w, s.d, "register: hash password", err)
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, s.d, "register: begin transaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO users (email, password_hash, username, display_name, email_verified, is_active)
		VALUES (?, ?, ?, ?, 0, 1)`, email, hash, username, displayName)
	if err != nil {
		if code := conflictCode(err); code != "" {
			if code == codeEmailTaken {
				shared.WriteError(w, http.StatusConflict, code, "An account with that email already exists.")
			} else {
				shared.WriteError(w, http.StatusConflict, code, "That username is already taken.")
			}
			return
		}
		internalError(w, s.d, "register: insert user", err)
		return
	}
	userID, err := result.LastInsertId()
	if err != nil {
		internalError(w, s.d, "register: last insert id", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`INSERT INTO user_profiles (user_id) VALUES (?)`, userID); err != nil {
		internalError(w, s.d, "register: insert profile", err)
		return
	}

	verificationToken, err := newToken()
	if err != nil {
		internalError(w, s.d, "register: verification token", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO email_verification_tokens (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+1 day'))`, hashToken(verificationToken), userID); err != nil {
		internalError(w, s.d, "register: store verification token", err)
		return
	}

	sessionToken, err := newToken()
	if err != nil {
		internalError(w, s.d, "register: session token", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+7 days'))`, hashToken(sessionToken), userID); err != nil {
		internalError(w, s.d, "register: store session", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, s.d, "register: commit", err)
		return
	}

	user, err := s.userByID(r.Context(), userID)
	if err != nil {
		internalError(w, s.d, "register: load user", err)
		return
	}
	setSessionCookie(w, s.d, sessionToken)
	response := registerResponse{User: user}
	if !s.d.Config.IsProduction() {
		response.DevVerificationToken = verificationToken
	}
	shared.WriteJSON(w, http.StatusCreated, response)
}

func (s *service) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	details := make(validationErrors)
	email := validateEmail(in.Email, details)
	if in.Password == "" {
		details.add("password", "Password is required.")
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	var userID int64
	var hash string
	var active int
	err := s.d.DB.QueryRowContext(r.Context(),
		`SELECT id, password_hash, is_active FROM users WHERE email = ? LIMIT 1`, email).
		Scan(&userID, &hash, &active)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Do a comparable amount of work so the response cannot reveal whether
		// the email is registered.
		_ = shared.CheckPassword(dummyHash, in.Password)
		shared.WriteError(w, http.StatusUnauthorized, codeInvalidCredentials, "Invalid email or password.")
		return
	case err != nil:
		internalError(w, s.d, "login: lookup user", err)
		return
	}
	if active != 1 || !shared.CheckPassword(hash, in.Password) {
		shared.WriteError(w, http.StatusUnauthorized, codeInvalidCredentials, "Invalid email or password.")
		return
	}

	token, err := issueSession(r.Context(), s.d.DB, userID)
	if err != nil {
		internalError(w, s.d, "login: issue session", err)
		return
	}
	user, err := s.userByID(r.Context(), userID)
	if err != nil {
		internalError(w, s.d, "login: load user", err)
		return
	}
	setSessionCookie(w, s.d, token)
	shared.WriteJSON(w, http.StatusOK, user)
}

func (s *service) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	user, err := s.userByID(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if err != nil {
		internalError(w, s.d, "me: load user", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, user)
}

func (s *service) logout(w http.ResponseWriter, r *http.Request) {
	if _, ok := shared.UserID(r); !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if cookie, err := r.Cookie(shared.SessionCookieName); err == nil && cookie.Value != "" {
		if _, err := s.d.DB.ExecContext(r.Context(),
			`DELETE FROM sessions WHERE token_hash = ?`, hashToken(cookie.Value)); err != nil {
			internalError(w, s.d, "logout: delete session", err)
			return
		}
	}
	clearSessionCookie(w, s.d)
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) verify(w http.ResponseWriter, r *http.Request) {
	var in tokenRequest
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Token) == "" {
		writeValidation(w, validationErrors{"token": "Verification token is required."})
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, s.d, "verify: begin transaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	err = tx.QueryRowContext(r.Context(), `
		SELECT user_id FROM email_verification_tokens
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > datetime('now')
		LIMIT 1`, hashToken(in.Token)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidToken, "This verification link is invalid or has expired.")
		return
	}
	if err != nil {
		internalError(w, s.d, "verify: lookup token", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE users SET email_verified = 1, updated_at = datetime('now') WHERE id = ?`, userID); err != nil {
		internalError(w, s.d, "verify: mark email verified", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE email_verification_tokens SET used_at = datetime('now') WHERE token_hash = ?`, hashToken(in.Token)); err != nil {
		internalError(w, s.d, "verify: consume token", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, s.d, "verify: commit", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) passwordResetRequest(w http.ResponseWriter, r *http.Request) {
	var in passwordResetRequest
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	details := make(validationErrors)
	email := validateEmail(in.Email, details)
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	response := passwordResetResponse{
		Message: "If that email is registered, a password reset link has been sent.",
	}
	var userID int64
	err := s.d.DB.QueryRowContext(r.Context(),
		`SELECT id FROM users WHERE email = ? AND is_active = 1 LIMIT 1`, email).Scan(&userID)
	if err == nil {
		token, tokenErr := newToken()
		if tokenErr != nil {
			internalError(w, s.d, "reset request: token", tokenErr)
			return
		}
		_, err = s.d.DB.ExecContext(r.Context(), `
			INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
			VALUES (?, ?, datetime('now', '+30 minutes'))`, hashToken(token), userID)
		if err != nil {
			internalError(w, s.d, "reset request: store token", err)
			return
		}
		if !s.d.Config.IsProduction() {
			response.DevResetToken = token
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		internalError(w, s.d, "reset request: lookup user", err)
		return
	}
	// Same status and message whether or not the address exists.
	shared.WriteJSON(w, http.StatusAccepted, response)
}

func (s *service) passwordResetConfirm(w http.ResponseWriter, r *http.Request) {
	var in passwordResetConfirmRequest
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	details := make(validationErrors)
	if strings.TrimSpace(in.Token) == "" {
		details.add("token", "Reset token is required.")
	}
	validatePassword(in.Password, details)
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, s.d, "reset confirm: begin transaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	err = tx.QueryRowContext(r.Context(), `
		SELECT user_id FROM password_reset_tokens
		WHERE token_hash = ? AND used_at IS NULL AND expires_at > datetime('now')
		LIMIT 1`, hashToken(in.Token)).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidToken, "This reset link is invalid or has expired.")
		return
	}
	if err != nil {
		internalError(w, s.d, "reset confirm: lookup token", err)
		return
	}
	hash, err := shared.HashPassword(in.Password)
	if err != nil {
		internalError(w, s.d, "reset confirm: hash password", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?`, hash, userID); err != nil {
		internalError(w, s.d, "reset confirm: update password", err)
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE password_reset_tokens SET used_at = datetime('now') WHERE token_hash = ?`, hashToken(in.Token)); err != nil {
		internalError(w, s.d, "reset confirm: consume token", err)
		return
	}
	// A password change revokes every existing session for the account.
	if _, err := tx.ExecContext(r.Context(),
		`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		internalError(w, s.d, "reset confirm: revoke sessions", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, s.d, "reset confirm: commit", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
