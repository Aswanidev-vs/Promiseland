package events

import (
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// notifyUser is deliberately optimistic: notifications are owned by another
// module and may not have the expected schema yet. A mismatch must never fail
// the RSVP or update that triggered it.
func notifyUser(d *shared.Deps, userID, actorID int64, notificationType, targetType string, targetID int64) {
	if d == nil || d.DB == nil || userID <= 0 || userID == actorID || !tableExists(d, "notifications") {
		return
	}
	_, err := d.DB.Exec(`
		INSERT INTO notifications(user_id, type, actor_id, target_type, target_id, created_at)
		VALUES(?,?,?,?,?,datetime('now'))`, userID, notificationType, actorID, targetType, targetID)
	if err != nil {
		if d.Log != nil {
			d.Log.Warn("notification insert skipped",
				"error", err, "type", notificationType, "target_id", targetID)
		}
	}
}

// notifyAttendees tells current RSVP holders about a new organiser update.
func (h *handler) notifyAttendees(eventID, actorID int64) {
	if h.d == nil || h.d.DB == nil || !tableExists(h.d, "event_rsvps") {
		return
	}
	rows, err := h.d.DB.Query(`SELECT user_id FROM event_rsvps WHERE event_id = ?`, eventID)
	if err != nil {
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		notifyUser(h.d, id, actorID, "event_update", "event", eventID)
	}
}
