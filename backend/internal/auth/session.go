package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const sessionLifetime = 7 * 24 * time.Hour

// newToken returns a URL-safe, 256-bit opaque token. Raw tokens are only ever
// sent to the client; the database stores their SHA-256 digest.
func newToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func issueSession(ctx context.Context, db *sql.DB, userID int64) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES (?, ?, datetime('now', '+7 days'))`, hashToken(token), userID)
	if err != nil {
		return "", err
	}
	return token, nil
}

func setSessionCookie(w http.ResponseWriter, d *shared.Deps, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     shared.SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   d.Config.IsProduction(),
		MaxAge:   int(sessionLifetime / time.Second), // 604800
		Expires:  time.Now().Add(sessionLifetime),
	})
}

func clearSessionCookie(w http.ResponseWriter, d *shared.Deps) {
	http.SetCookie(w, &http.Cookie{
		Name:     shared.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   d.Config.IsProduction(),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
}
