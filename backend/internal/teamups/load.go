package teamups

import (
	"context"
	"database/sql"
	"fmt"
)

// teamUpSelect is reused by every read path so applicant_count and JSON
// serialization are identical everywhere.
const teamUpSelect = `
SELECT t.id, t.owner_id, t.title, t.description, t.event_id, t.categories,
       t.status,
       (SELECT COUNT(*) FROM team_up_applications a WHERE a.team_up_id = t.id),
       t.created_at
FROM team_ups t`

type teamUpRow struct {
	ID             int64
	OwnerID        int64
	Title          string
	Description    string
	EventID        sql.NullInt64
	Categories     string
	Status         string
	ApplicantCount int
	CreatedAt      string
}

func scanTeamUpRows(rows *sql.Rows) ([]teamUpRow, error) {
	out := []teamUpRow{}
	for rows.Next() {
		var row teamUpRow
		if err := rows.Scan(&row.ID, &row.OwnerID, &row.Title, &row.Description,
			&row.EventID, &row.Categories, &row.Status, &row.ApplicantCount,
			&row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *service) loadUsers(ctx context.Context, ids []int64) map[int64]UserSummary {
	out := make(map[int64]UserSummary, len(ids))
	if len(ids) == 0 || !s.schema.HasTable("users") {
		return out
	}
	username := "''"
	switch {
	case s.schema.HasCol("users", "username"):
		username = "COALESCE(NULLIF(u.username, ''), u.display_name, '')"
	case s.schema.HasCol("user_profiles", "username"):
		username = "COALESCE(up.username, '')"
	}
	display := "COALESCE(u.display_name, '')"
	if !s.schema.HasCol("users", "display_name") && s.schema.HasCol("user_profiles", "display_name") {
		display = "COALESCE(up.display_name, '')"
	}
	avatar := "CAST(NULL AS TEXT)"
	if s.schema.HasCol("user_profiles", "avatar_url") {
		avatar = "up.avatar_url"
	} else if s.schema.HasCol("users", "avatar_url") {
		avatar = "u.avatar_url"
	}
	q := fmt.Sprintf(`SELECT u.id, %s, %s, %s
		FROM users u LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE u.id IN (%s)`, username, display, avatar, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var user UserSummary
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.AvatarURL); err == nil {
			out[user.ID] = user
		}
	}
	return out
}

func (s *service) buildTeamUps(ctx context.Context, rows []teamUpRow, viewerID int64) ([]TeamUp, error) {
	out := make([]TeamUp, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ownerIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		ownerIDs = append(ownerIDs, row.OwnerID)
	}
	owners := s.loadUsers(ctx, ownerIDs)

	for _, row := range rows {
		item := TeamUp{
			ID:             row.ID,
			Owner:          owners[row.OwnerID],
			Title:          row.Title,
			Description:    row.Description,
			Categories:     decodeCategories(row.Categories),
			Status:         row.Status,
			ApplicantCount: row.ApplicantCount,
			CreatedAt:      row.CreatedAt,
		}
		if item.Owner.ID == 0 {
			item.Owner = UserSummary{ID: row.OwnerID, DisplayName: "unknown"}
		}
		if row.EventID.Valid {
			item.Event = s.loadEvent(ctx, row.EventID.Int64, viewerID)
		}
		if viewerID > 0 && s.schema.HasTable("team_up_applications") {
			var status string
			if err := s.d.DB.QueryRowContext(ctx, `
				SELECT status FROM team_up_applications
				WHERE team_up_id = ? AND applicant_id = ? LIMIT 1`,
				row.ID, viewerID).Scan(&status); err == nil {
				value := status
				item.MyApplication = &value
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// loadEvent is defensive: it only queries columns the events migration is
// known to provide and returns nil if the table is absent or still evolving.
func (s *service) loadEvent(ctx context.Context, id, viewerID int64) *Event {
	sc := s.schema
	if !sc.HasTable("events") {
		return nil
	}
	required := []string{
		"id", "organizer_id", "title", "description", "event_type",
		"starts_at", "ends_at", "timezone", "venue_text", "city", "country",
		"is_online", "created_at",
	}
	for _, column := range required {
		if !sc.HasCol("events", column) {
			return nil
		}
	}
	text := func(column string) string {
		return "COALESCE(CAST(e." + column + " AS TEXT), '')"
	}
	nullableText := func(column string) string {
		if sc.HasCol("events", column) {
			return "e." + column
		}
		return "CAST(NULL AS TEXT)"
	}
	integer := func(column string) string {
		if sc.HasCol("events", column) {
			return "COALESCE(e." + column + ", 0)"
		}
		return "0"
	}
	rsvpCount := "0"
	rsvpStatus := "CAST(NULL AS TEXT)"
	queryArgs := []any{}
	if sc.HasTable("event_rsvps") && sc.HasCol("event_rsvps", "event_id") {
		rsvpCount = "(SELECT COUNT(*) FROM event_rsvps er WHERE er.event_id = e.id)"
		if viewerID > 0 && sc.HasCol("event_rsvps", "user_id") && sc.HasCol("event_rsvps", "status") {
			rsvpStatus = `(SELECT er.status FROM event_rsvps er
				WHERE er.event_id = e.id AND er.user_id = ? LIMIT 1)`
			queryArgs = append(queryArgs, viewerID)
		}
	}
	cover := nullableText("cover_url")
	if sc.HasCol("events", "cover_media_id") && sc.HasTable("media") && sc.HasCol("media", "storage_key") {
		// The events module stores a media id; expose the same /media/ URL that
		// the events API produces so the embedded Event JSON stays consistent.
		cover = `(SELECT '/media/' || COALESCE(m.storage_key, '')
			FROM media m WHERE m.id = e.cover_media_id)`
	}
	q := fmt.Sprintf(`SELECT e.id, e.organizer_id, %s, %s, %s, %s, %s, %s, %s, %s, %s,
		CASE WHEN COALESCE(e.is_online, 0) != 0 THEN 1 ELSE 0 END,
		%s, %s, %s, %s, %s, %s
		FROM events e WHERE e.id = ?`,
		text("title"), text("description"), text("event_type"),
		text("starts_at"), text("ends_at"), text("timezone"),
		text("venue_text"), text("city"), text("country"),
		integer("capacity"), rsvpCount, cover, rsvpStatus,
		nullableText("status"), text("created_at"))
	// rsvpStatus (when present) is bound before the final event-id predicate.
	queryArgs = append(queryArgs, id)
	var event Event
	var organizerID int64
	var online int
	var coverURL, status, myRSVP sql.NullString
	if err := s.d.DB.QueryRowContext(ctx, q, queryArgs...).Scan(
		&event.ID, &organizerID, &event.Title, &event.Description, &event.EventType,
		&event.StartsAt, &event.EndsAt, &event.Timezone, &event.VenueText,
		&event.City, &event.Country, &online, &event.Capacity, &event.RSVPCount,
		&coverURL, &myRSVP, &status, &event.CreatedAt,
	); err != nil {
		return nil
	}
	event.IsOnline = online == 1
	event.CoverURL = nullString(coverURL)
	event.RSVPStatus = nullString(myRSVP)
	event.Status = status.String
	event.Organizer = s.loadUsers(ctx, []int64{organizerID})[organizerID]
	if event.Organizer.ID == 0 {
		event.Organizer = UserSummary{ID: organizerID, DisplayName: "unknown"}
	}
	return &event
}

// notify writes an optimistic in-app notification. Missing tables or schema
// differences are logged and ignored so the primary request always succeeds.
func (s *service) notify(userID, actorID int64, kind, targetType string, targetID int64, data string) {
	if s == nil || s.d == nil || s.d.DB == nil || !s.schema.HasTable("notifications") || userID <= 0 {
		return
	}
	_, err := s.d.DB.Exec(`
		INSERT INTO notifications(user_id, type, actor_id, target_type, target_id, data, created_at)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'))`,
		userID, kind, actorID, targetType, targetID, data)
	if err != nil && s.d.Log != nil {
		s.d.Log.Warn("teamups: could not write notification",
			"error", err, "type", kind, "target_id", targetID)
	}
}
