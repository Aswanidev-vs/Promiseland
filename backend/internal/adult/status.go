package adult

import (
	"net/http"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// handleStatus is the only unauthenticated route in the namespace. It is
// explicitly safe: when the feature is disabled it returns the fixed
// disabled shape with HTTP 404 and no eligibility or profile information.
func (m *module) handleStatus(w http.ResponseWriter, r *http.Request) {
	if !m.enabled() {
		shared.WriteJSON(w, http.StatusNotFound, AdultStatus{
			Enabled:      false,
			AgeAssurance: assuranceNone,
			Eligible:     false,
			Message:      disabledMessage,
		})
		return
	}

	// A sentinel user id is enough to ask the configured implementation for
	// its mode string; deny-all and the development stub both answer without
	// revealing anything about a particular account.
	_, mode := m.assurance.Eligible(0)
	eligible := false
	if userID, ok := m.viewer(r); ok && m.d.DB != nil {
		eligible, _ = m.assurance.Eligible(userID)
	}

	message := "Age assurance is required before you can use this area."
	if eligible {
		message = "You can use this adults-only area."
	} else if mode == assuranceNone {
		message = "Age assurance is not configured; this area is unavailable."
	}
	shared.WriteJSON(w, http.StatusOK, AdultStatus{
		Enabled:      true,
		AgeAssurance: mode,
		Eligible:     eligible,
		Message:      message,
	})
}
