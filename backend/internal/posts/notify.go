package posts

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_-]{2,30})`)

// notifyUser is deliberately optimistic: notifications are owned by another
// agent and may not be migrated yet. A missing table or schema mismatch must
// never fail the post or comment that triggered it.
func notifyUser(d *shared.Deps, userID, actorID int64, notificationType, targetType string, targetID int64) {
	if d == nil || d.DB == nil || userID <= 0 || userID == actorID {
		return
	}
	if !tableExists(d, "notifications") {
		return
	}
	_, err := d.DB.Exec(`
		INSERT INTO notifications(user_id, type, actor_id, target_type, target_id, created_at)
		VALUES(?,?,?,?,?,datetime('now'))`, userID, notificationType, actorID, targetType, targetID)
	if err != nil {
		if d.Log != nil {
			d.Log.Warn("notification insert skipped", "error", err, "type", notificationType)
		}
	}
}

func (h *handler) notifyMentions(actorID int64, body, targetType string, targetID int64) {
	matches := mentionPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return
	}
	names := make([]string, 0, len(matches))
	args := make([]any, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		name := strings.ToLower(match[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
		args = append(args, name)
	}
	blocked := map[int64]bool{}
	if tableExists(h.d, "blocks") {
		rows, err := h.d.DB.Query(`
			SELECT blocker_id, blocked_id FROM blocks WHERE blocker_id = ? OR blocked_id = ?`,
			actorID, actorID)
		if err == nil {
			for rows.Next() {
				var blocker, blockedID int64
				if err := rows.Scan(&blocker, &blockedID); err == nil {
					if blocker == actorID {
						blocked[blockedID] = true
					} else {
						blocked[blocker] = true
					}
				}
			}
			rows.Close()
		}
	}
	column := "username"
	if !columnExists(h.d, "users", "username") {
		column = "display_name"
	}
	query := fmt.Sprintf(`SELECT id FROM users WHERE LOWER(%s) IN (%s)`, column, placeholders(len(names)))
	rows, err := h.d.DB.Query(query, args...)
	if err != nil {
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil && !blocked[id] && id != actorID {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		notifyUser(h.d, id, actorID, "mention", targetType, targetID)
	}
}
