// Package shared contains cross-cutting types and helpers used by every
// domain module. Domain packages must not import each other.
package shared

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/config"
)

// SessionCookieName is the HTTP-only cookie carrying the opaque session token.
const SessionCookieName = "pl_session"

// Deps bundles the process-wide dependencies passed to every module's
// Register function.
type Deps struct {
	DB     *sql.DB
	Log    *slog.Logger
	Config config.Config
}

// SessionUserID resolves the signed-in user from the pl_session cookie. The
// token is hashed with SHA-256 before lookup, so raw tokens are never used in
// queries or logs. Expired sessions are rejected.
func (d *Deps) SessionUserID(r *http.Request) (int64, bool) {
	if d == nil || d.DB == nil {
		return 0, false
	}
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return 0, false
	}

	sum := sha256.Sum256([]byte(cookie.Value))
	tokenHash := hex.EncodeToString(sum[:])

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	var userID int64
	// A session only counts while its account is active: suspending someone
	// must end the access they already hold, not wait for the cookie to expire.
	err = d.DB.QueryRowContext(ctx, `
		SELECT s.user_id
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > datetime('now') AND u.is_active = 1
		LIMIT 1`, tokenHash).Scan(&userID)
	if err != nil {
		// Missing, expired or revoked sessions are all simply unauthenticated.
		return 0, false
	}
	return userID, true
}
