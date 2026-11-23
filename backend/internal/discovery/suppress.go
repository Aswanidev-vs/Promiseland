package discovery

import (
	"net/http"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

func (m *module) handleSuppress(w http.ResponseWriter, r *http.Request) {
	m.setSuppression(w, r, true)
}

func (m *module) handleUnsuppress(w http.ResponseWriter, r *http.Request) {
	m.setSuppression(w, r, false)
}

func (m *module) setSuppression(w http.ResponseWriter, r *http.Request, suppressed bool) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	targetID, ok := parsePositiveID(r, "userId")
	if !ok {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid user id.")
		return
	}
	if targetID == viewerID {
		writeValidation(w, "Invalid user id.", fieldErrors{
			"userId": "you cannot suppress yourself",
		})
		return
	}
	exists, err := m.userExists(r.Context(), targetID)
	if err != nil {
		internalError(w, m.d, "suppress: check target", err)
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}

	if suppressed {
		_, err = m.d.DB.ExecContext(r.Context(), `
			INSERT OR IGNORE INTO discovery_suppressions (viewer_id, candidate_id)
			VALUES (?, ?)`, viewerID, targetID)
	} else {
		_, err = m.d.DB.ExecContext(r.Context(), `
			DELETE FROM discovery_suppressions
			WHERE viewer_id = ? AND candidate_id = ?`, viewerID, targetID)
	}
	if err != nil {
		internalError(w, m.d, "suppress: update state", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
