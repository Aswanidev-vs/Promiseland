package auth

import (
	"database/sql"
	"regexp"
	"strings"
	"unicode/utf8"
)

// User is the exact R1 wire shape from docs/conventions.md. Email is only
// returned by self-facing auth endpoints; public profile DTOs live in users.
type User struct {
	ID           int64   `json:"id"`
	Email        string  `json:"email"`
	Username     string  `json:"username"`
	DisplayName  string  `json:"display_name"`
	AvatarURL    *string `json:"avatar_url"`
	BannerURL    *string `json:"banner_url"`
	Bio          *string `json:"bio"`
	Location     *string `json:"location"`
	Visibility   string  `json:"visibility"`
	Discoverable bool    `json:"discoverable"`
	CreatedAt    string  `json:"created_at"`
	/** True when this account holds a platform staff grant (the admin console). */
	IsStaff bool `json:"is_staff"`
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type registerResponse struct {
	User
	// DevVerificationToken is present only when APP_ENV != production. It
	// exists so local clients and tests can verify an address without an email
	// provider. Never rely on it in production.
	DevVerificationToken string `json:"devVerificationToken,omitempty"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenRequest struct {
	Token string `json:"token"`
}

type passwordResetRequest struct {
	Email string `json:"email"`
}

type passwordResetResponse struct {
	Message       string `json:"message"`
	DevResetToken string `json:"devResetToken,omitempty"`
}

type passwordResetConfirmRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

const (
	maxEmailLen         = 254
	maxPasswordBytes    = 72 // bcrypt truncates beyond this; reject rather than silently ignore
	maxDisplayNameRunes = 50
)

var (
	emailRE    = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	usernameRE = regexp.MustCompile(`^[a-z0-9_]{3,20}$`)
)

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// validationErrors is the details object used by WriteErrorDetails:
// {"field": "message"}.
type validationErrors map[string]string

func (v validationErrors) add(field, message string) {
	if _, exists := v[field]; !exists {
		v[field] = message
	}
}

func validateEmail(value string, details validationErrors) string {
	email := normalizeEmail(value)
	switch {
	case email == "":
		details.add("email", "Email is required.")
	case len(email) > maxEmailLen:
		details.add("email", "Email must be at most 254 characters.")
	case !emailRE.MatchString(email):
		details.add("email", "Enter a valid email address.")
	}
	return email
}

func validatePassword(value string, details validationErrors) {
	switch {
	case value == "":
		details.add("password", "Password is required.")
	case len(value) < 8:
		details.add("password", "Password must be at least 8 characters.")
	case len(value) > maxPasswordBytes:
		details.add("password", "Password must be at most 72 bytes.")
	}
}

func validateUsername(value string, details validationErrors) string {
	username := normalizeUsername(value)
	if !usernameRE.MatchString(username) {
		details.add("username", "Username must be 3-20 characters: lowercase letters, numbers and underscores.")
	}
	return username
}

func validateDisplayName(value string, details validationErrors) string {
	name := strings.TrimSpace(value)
	if name == "" {
		details.add("displayName", "Display name is required.")
	} else if utf8.RuneCountInString(name) > maxDisplayNameRunes {
		details.add("displayName", "Display name must be at most 50 characters.")
	}
	return name
}

func stringPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	value := ns.String
	return &value
}
