package adult

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const (
	maxDisplayNameRunes = 80
	maxBioRunes         = 2000
	maxBoundariesRunes  = 1000
)

var validIntents = map[string]bool{
	"friendship": true,
	"dating":     true,
	"both":       true,
}

type profileInput struct {
	DisplayName      string `json:"display_name"`
	CamelDisplayName string `json:"displayName"`
	Intent           string `json:"intent"`
	Bio              string `json:"bio"`
	Boundaries       string `json:"boundaries"`
}

func (in profileInput) displayName() string {
	if strings.TrimSpace(in.CamelDisplayName) != "" {
		return strings.TrimSpace(in.CamelDisplayName)
	}
	return strings.TrimSpace(in.DisplayName)
}

func (m *module) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	profile, found, err := m.loadProfile(r.Context(), userID)
	if err != nil {
		internalError(w, m.d, "load adult profile", err)
		return
	}
	if !found {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Adult profile not found.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, profile)
}

func (m *module) handlePutProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	var body profileInput
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	displayName := body.displayName()
	intent := strings.ToLower(strings.TrimSpace(body.Intent))
	bio := strings.TrimSpace(body.Bio)
	boundaries := strings.TrimSpace(body.Boundaries)

	details := map[string]string{}
	if displayName == "" || utf8.RuneCountInString(displayName) > maxDisplayNameRunes {
		details["display_name"] = "must be between 1 and 80 characters"
	}
	if !validIntents[intent] {
		details["intent"] = "must be friendship, dating or both"
	}
	if utf8.RuneCountInString(bio) > maxBioRunes {
		details["bio"] = "must be at most 2000 characters"
	}
	if utf8.RuneCountInString(boundaries) > maxBoundariesRunes {
		details["boundaries"] = "must be at most 1000 characters"
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	_, err := m.d.DB.ExecContext(r.Context(), `
		INSERT INTO adult_profiles (user_id, display_name, intent, bio, boundaries, visibility, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'opt_in', datetime('now'), NULL)
		ON CONFLICT(user_id) DO UPDATE SET
			display_name = excluded.display_name,
			intent = excluded.intent,
			bio = excluded.bio,
			boundaries = excluded.boundaries,
			updated_at = datetime('now')`,
		userID, displayName, intent, bio, boundaries)
	if err != nil {
		internalError(w, m.d, "save adult profile", err)
		return
	}
	profile, found, err := m.loadProfile(r.Context(), userID)
	if err != nil || !found {
		internalError(w, m.d, "reload adult profile", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, profile)
}

func (m *module) loadProfile(ctx context.Context, userID int64) (AdultProfile, bool, error) {
	var p AdultProfile
	if m == nil || m.d == nil || m.d.DB == nil {
		return p, false, nil
	}
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT user_id, display_name, intent, bio, boundaries, visibility, created_at
		FROM adult_profiles WHERE user_id = ?`, userID).
		Scan(&p.UserID, &p.DisplayName, &p.Intent, &p.Bio, &p.Boundaries, &p.Visibility, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AdultProfile{}, false, nil
	}
	if err != nil {
		return AdultProfile{}, false, err
	}
	return p, true, nil
}

// blockedEitherWay reports whether either account has blocked the other.
func blockedEitherWay(ctx context.Context, db *sql.DB, a, b int64) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx, `
		SELECT 1 FROM blocks
		WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)
		LIMIT 1`, a, b, b, a).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
