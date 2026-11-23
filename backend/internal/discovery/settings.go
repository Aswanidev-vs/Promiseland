package discovery

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// loadDiscoverable reads the discoverability flag. The flag lives on users in
// the current schema (001_auth_users.sql); user_profiles owns visibility.
func (m *module) loadDiscoverable(ctx context.Context, userID int64) (bool, error) {
	var value int
	err := m.d.DB.QueryRowContext(ctx,
		`SELECT discoverable FROM users WHERE id = ? LIMIT 1`, userID).Scan(&value)
	return value != 0, err
}

func (m *module) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	discoverable, err := m.loadDiscoverable(r.Context(), viewerID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if err != nil {
		internalError(w, m.d, "load discovery settings", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, DiscoverySettings{Discoverable: discoverable})
}

func (m *module) handlePatchSettings(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	raw, ok := decodeFields(w, r)
	if !ok {
		return
	}
	value, valid := decodeBool(raw["discoverable"])
	if !valid {
		writeValidation(w, "Invalid discovery settings.", fieldErrors{
			"discoverable": "must be a boolean",
		})
		return
	}
	flag := 0
	if value {
		flag = 1
	}
	result, err := m.d.DB.ExecContext(r.Context(),
		`UPDATE users SET discoverable = ?, updated_at = datetime('now') WHERE id = ?`, flag, viewerID)
	if err != nil {
		internalError(w, m.d, "update discovery settings", err)
		return
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, DiscoverySettings{Discoverable: value})
}
