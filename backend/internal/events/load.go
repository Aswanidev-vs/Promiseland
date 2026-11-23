package events

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const eventSelect = `
SELECT e.id, e.organizer_id, e.title, e.description, e.event_type,
       e.starts_at, e.ends_at, e.timezone,
       COALESCE(e.venue_text, ''), COALESCE(e.city, ''), COALESCE(e.country, ''),
       e.is_online, e.capacity, e.cover_media_id, e.status, e.created_at,
       (SELECT COUNT(*) FROM event_rsvps r
          WHERE r.event_id = e.id AND r.status = 'going'),
       (SELECT r.status FROM event_rsvps r
          WHERE r.event_id = e.id AND r.user_id = ?)
FROM events e`

func scanEvent(rows *sql.Rows) (eventRow, error) {
	var row eventRow
	var online int
	var rsvpStatus sql.NullString
	err := rows.Scan(
		&row.ID, &row.OrganizerID, &row.Title, &row.Description, &row.EventType,
		&row.StartsAt, &row.EndsAt, &row.Timezone,
		&row.VenueText, &row.City, &row.Country,
		&online, &row.Capacity, &row.CoverMediaID, &row.Status, &row.CreatedAt,
		&row.RSVPCount, &rsvpStatus,
	)
	if err != nil {
		return row, err
	}
	row.IsOnline = online == 1
	row.RSVPStatus = nullString(rsvpStatus)
	return row, nil
}

func eventByID(ctx context.Context, d *shared.Deps, id, viewer int64) (eventRow, error) {
	var row eventRow
	rows, err := d.DB.QueryContext(ctx, eventSelect+` WHERE e.id = ?`, viewer, id)
	if err != nil {
		return row, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return row, err
		}
		return row, sql.ErrNoRows
	}
	return scanEvent(rows)
}

// visibleEvent returns false when the event does not exist or is cancelled and
// the caller is not its organiser.
func visibleEvent(row eventRow, viewer int64) bool {
	return row.Status != "cancelled" || row.OrganizerID == viewer
}

func loadRawEvent(ctx context.Context, d *shared.Deps, id int64) (eventFacts, error) {
	var e eventFacts
	err := d.DB.QueryRowContext(ctx, `
		SELECT id, organizer_id, status, capacity, starts_at, ends_at
		FROM events WHERE id = ?`, id).
		Scan(&e.ID, &e.OrganizerID, &e.Status, &e.Capacity, &e.StartsAt, &e.EndsAt)
	return e, err
}

// buildEvents expands raw rows into the exact R2 event JSON shape. User and
// cover lookups are batched to avoid an N+1 pattern on list responses.
func buildEvents(ctx context.Context, d *shared.Deps, rows []eventRow) []Event {
	out := make([]Event, 0, len(rows))
	if len(rows) == 0 {
		return out
	}
	authorIDs := make([]int64, 0, len(rows))
	mediaIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.OrganizerID)
		if row.CoverMediaID.Valid && row.CoverMediaID.Int64 > 0 {
			mediaIDs = append(mediaIDs, row.CoverMediaID.Int64)
		}
	}
	authors := loadUsers(ctx, d, authorIDs)
	covers := loadCovers(ctx, d, mediaIDs)

	for _, row := range rows {
		item := row.Event
		item.Organizer = authors[row.OrganizerID]
		if item.Organizer.ID == 0 {
			item.Organizer = UserSummary{ID: row.OrganizerID, DisplayName: "unknown"}
		}
		item.CoverURL = nil
		if row.CoverMediaID.Valid {
			if url, ok := covers[row.CoverMediaID.Int64]; ok {
				item.CoverURL = &url
			}
		}
		out = append(out, item)
	}
	return out
}

func loadCovers(ctx context.Context, d *shared.Deps, ids []int64) map[int64]string {
	out := map[int64]string{}
	if len(ids) == 0 || !tableExists(d, "media") || !columnExists(d, "media", "storage_key") {
		return out
	}
	q := fmt.Sprintf(`SELECT id, storage_key FROM media WHERE id IN (%s)`, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var key string
		if err := rows.Scan(&id, &key); err == nil {
			out[id] = "/media/" + key
		}
	}
	return out
}

func loadUsers(ctx context.Context, d *shared.Deps, ids []int64) map[int64]UserSummary {
	out := make(map[int64]UserSummary, len(ids))
	if len(ids) == 0 {
		return out
	}
	usernameExpr := "NULLIF(u.username, '')"
	if !columnExists(d, "users", "username") {
		usernameExpr = "NULLIF(u.display_name, '')"
	}
	q := fmt.Sprintf(`
		SELECT u.id, COALESCE(%s, u.display_name, ''), u.display_name, p.avatar_url
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id IN (%s)`, usernameExpr, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var u UserSummary
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.AvatarURL); err == nil {
			out[u.ID] = u
		}
	}
	return out
}
