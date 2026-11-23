package adult

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const maxReportDetailsRunes = 2000

// validReportReasons mirrors the existing moderation reason codes so adult
// safety reports land in the same triage vocabulary as other reports.
var validReportReasons = map[string]bool{
	"spam": true, "harassment": true, "impersonation": true,
	"hate_abuse": true, "inappropriate": true, "copyright": true,
	"spoiler_abuse": true, "unsafe_event": true, "other": true,
}

type reportInput struct {
	TargetType string `json:"targetType"`
	TargetID   int64  `json:"targetId"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

// handleReport writes a high-priority row to the EXISTING reports table. The
// target type is always 'adult_profile' regardless of client input, and the
// details payload marks it for the prioritized dating-safety workflow.
func (m *module) handleReport(w http.ResponseWriter, r *http.Request) {
	reporterID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	var body reportInput
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	reason := strings.ToLower(strings.TrimSpace(body.Reason))
	details := strings.TrimSpace(body.Details)

	fieldErrors := map[string]string{}
	if body.TargetID <= 0 {
		fieldErrors["targetId"] = "must be a positive integer"
	}
	if !validReportReasons[reason] {
		fieldErrors["reason"] = "must be an existing moderation reason code"
	}
	if utf8.RuneCountInString(details) > maxReportDetailsRunes {
		fieldErrors["details"] = "must be at most 2000 characters"
	}
	if len(fieldErrors) > 0 {
		writeValidation(w, fieldErrors)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	// Only an actual opt-in adult profile can be reported through this route.
	visible, err := m.visibleTarget(ctx, body.TargetID)
	if err != nil {
		internalError(w, m.d, "check adult report target", err)
		return
	}
	if !visible {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Profile not found.")
		return
	}

	payload := map[string]string{"priority": "high", "context": "dating"}
	if details != "" {
		payload["note"] = details
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		internalError(w, m.d, "encode adult report details", err)
		return
	}

	result, err := m.d.DB.ExecContext(ctx, `
		INSERT INTO reports (reporter_id, target_type, target_id, reason, details, status)
		VALUES (?, 'adult_profile', ?, ?, ?, 'open')`,
		reporterID, body.TargetID, reason, string(encoded))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			shared.WriteError(w, http.StatusConflict, codeAlreadyReport,
				"You have already reported this profile.")
			return
		}
		internalError(w, m.d, "create adult report", err)
		return
	}
	id, _ := result.LastInsertId()

	var response reportResponse
	var storedDetails sql.NullString
	if err := m.d.DB.QueryRowContext(ctx, `
		SELECT id, reporter_id, target_type, target_id, reason, details, status, created_at
		FROM reports WHERE id = ?`, id).
		Scan(&response.ID, &response.ReporterID, &response.TargetType, &response.TargetID,
			&response.Reason, &storedDetails, &response.Status, &response.CreatedAt); err != nil {
		internalError(w, m.d, "reload adult report", err)
		return
	}
	if storedDetails.Valid {
		value := storedDetails.String
		response.Details = &value
	}
	shared.WriteJSON(w, http.StatusCreated, response)
}
