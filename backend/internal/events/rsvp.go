package events

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

var rsvpStatuses = map[string]bool{"going": true, "maybe": true}

// rsvp enforces capacity inside a transaction. "maybe" never occupies a slot,
// and switching from going to maybe releases one. Both this handler and
// cancelRSVP return the full updated Event (200), including refreshed
// rsvp_count and the caller's rsvp_status, as required by the R2 client.
func (h *handler) rsvp(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, ok := parseEventID(w, r)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	status := strings.TrimSpace(in.Status)
	if !rsvpStatuses[status] {
		writeValidation(w, map[string]string{"status": "Status must be going or maybe."})
		return
	}

	tx, err := h.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var organizerID int64
	var eventStatus string
	var capacity int
	err = tx.QueryRowContext(r.Context(),
		`SELECT organizer_id, status, capacity FROM events WHERE id = ?`, id).
		Scan(&organizerID, &eventStatus, &capacity)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
		return
	}
	if eventStatus == "cancelled" {
		shared.WriteError(w, http.StatusConflict, codeEventCancelled, "This event has been cancelled.")
		return
	}

	if status == "going" {
		var current string
		err := tx.QueryRowContext(r.Context(),
			`SELECT status FROM event_rsvps WHERE event_id = ? AND user_id = ?`, id, userID).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
			return
		}
		if errors.Is(err, sql.ErrNoRows) || current != "going" {
			var going int
			if err := tx.QueryRowContext(r.Context(),
				`SELECT COUNT(*) FROM event_rsvps WHERE event_id = ? AND status = 'going'`, id).Scan(&going); err != nil {
				shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
				return
			}
			if capacity > 0 && going >= capacity {
				shared.WriteError(w, http.StatusConflict, codeEventFull, "This event is full.")
				return
			}
		}
	}

	if _, err := tx.ExecContext(r.Context(), `
		INSERT INTO event_rsvps(event_id, user_id, status, created_at)
		VALUES(?,?,?,datetime('now'))
		ON CONFLICT(event_id, user_id) DO UPDATE SET status = excluded.status`,
		id, userID, status); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
		return
	}
	if err := tx.Commit(); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not save your RSVP.")
		return
	}

	notifyUser(h.d, organizerID, userID, "event_rsvp", "event", id)
	h.writeEvent(w, r, id, userID)
}

func (h *handler) cancelRSVP(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, ok := parseEventID(w, r)
	if !ok {
		return
	}
	_, err := loadRawEvent(r.Context(), h.d, id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	if _, err := h.d.DB.ExecContext(r.Context(),
		`DELETE FROM event_rsvps WHERE event_id = ? AND user_id = ?`, id, userID); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not withdraw your RSVP.")
		return
	}
	h.writeEvent(w, r, id, userID)
}

// writeEvent reloads the event after an RSVP change so the response always
// carries current rsvp_count and the caller's rsvp_status (never 204).
func (h *handler) writeEvent(w http.ResponseWriter, r *http.Request, id, viewer int64) {
	row, err := eventByID(r.Context(), h.d, id, viewer)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, buildEvents(r.Context(), h.d, []eventRow{row})[0])
}
