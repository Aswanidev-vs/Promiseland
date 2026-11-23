package adult

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// handleDevVerify is the ONLY writer of adult_verifications. It deliberately
// skips the eligibility check: with the deny-all default nobody could ever
// bootstrap the development stub. It requires a signed-in user, the enabled
// feature gate and APP_ENV=development; every other environment gets 404.
//
// This endpoint is NOT age assurance. It exists so the disabled-by-default
// scaffold can be exercised locally. A real provider must replace it before
// any launch; self-entered birthdays are never sufficient.
func (m *module) handleDevVerify(w http.ResponseWriter, r *http.Request) {
	if !devVerifyAvailable(m.d) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, disabledMessage)
		return
	}
	userID, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable,
			"The adults-only area is temporarily unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if _, err := m.d.DB.ExecContext(ctx, `
		INSERT INTO adult_verifications (user_id, status, method, verified_at, expires_at, created_at)
		VALUES (?, 'verified', 'development_stub', datetime('now'), datetime('now','+1 year'), datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET
			status = 'verified',
			method = 'development_stub',
			verified_at = datetime('now'),
			expires_at = datetime('now','+1 year')`, userID); err != nil {
		internalError(w, m.d, "development verification", err)
		return
	}

	var response verificationResponse
	var verifiedAt, expiresAt sql.NullString
	if err := m.d.DB.QueryRowContext(ctx, `
		SELECT user_id, status, method, verified_at, expires_at
		FROM adult_verifications WHERE user_id = ?`, userID).
		Scan(&response.UserID, &response.Status, &response.Method, &verifiedAt, &expiresAt); err != nil {
		internalError(w, m.d, "reload development verification", err)
		return
	}
	if verifiedAt.Valid {
		value := verifiedAt.String
		response.VerifiedAt = &value
	}
	if expiresAt.Valid {
		value := expiresAt.String
		response.ExpiresAt = &value
	}
	shared.WriteJSON(w, http.StatusOK, response)
}

func devVerifyAvailable(d *shared.Deps) bool {
	return d != nil && d.Config.FeatureAdultDating && isDevelopmentEnv(d.Config.AppEnv)
}
