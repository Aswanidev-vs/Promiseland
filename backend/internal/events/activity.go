package events

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

// subresourceEvent loads an event for a nested route and hides cancelled
// events from everyone except their organiser.
func (h *handler) subresourceEvent(w http.ResponseWriter, r *http.Request, viewer int64) (eventRow, bool) {
	id, ok := parseEventID(w, r)
	if !ok {
		return eventRow{}, false
	}
	row, err := eventByID(r.Context(), h.d, id, viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return eventRow{}, false
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return eventRow{}, false
	}
	if !visibleEvent(row, viewer) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return eventRow{}, false
	}
	return row, true
}

func (h *handler) updateByID(r *http.Request, id int64) (EventUpdate, error) {
	var u EventUpdate
	var authorID int64
	err := h.d.DB.QueryRowContext(r.Context(), `
		SELECT id, event_id, author_id, body, created_at FROM event_updates WHERE id = ?`, id).
		Scan(&u.ID, &u.EventID, &authorID, &u.Body, &u.CreatedAt)
	if err != nil {
		return u, err
	}
	users := loadUsers(r.Context(), h.d, []int64{authorID})
	u.Author = users[authorID]
	if u.Author.ID == 0 {
		u.Author = UserSummary{ID: authorID, DisplayName: "unknown"}
	}
	return u, nil
}

func (h *handler) listUpdates(w http.ResponseWriter, r *http.Request) {
	row, ok := h.subresourceEvent(w, r, viewerID(r, h.d))
	if !ok {
		return
	}
	rows, err := h.d.DB.QueryContext(r.Context(), `
		SELECT id FROM event_updates WHERE event_id = ? ORDER BY created_at DESC, id DESC`, row.ID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list event updates.")
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
	updates := make([]EventUpdate, 0, len(ids))
	for _, id := range ids {
		u, err := h.updateByID(r, id)
		if err == nil {
			updates = append(updates, u)
		}
	}
	shared.WriteJSON(w, http.StatusOK, updates)
}

// addUpdate returns the created EventUpdate JSON (201) for addEventUpdate.
func (h *handler) addUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	row, ok := h.subresourceEvent(w, r, userID)
	if !ok {
		return
	}
	if row.OrganizerID != userID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the organiser can publish updates.")
		return
	}
	var in struct {
		Body string `json:"body"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	body := strings.TrimSpace(in.Body)
	if body == "" || utf8.RuneCountInString(body) > maxBodyRunes {
		writeValidation(w, map[string]string{"body": "Update must be between 1 and 2000 characters."})
		return
	}
	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO event_updates(event_id, author_id, body, created_at)
		VALUES(?,?,?,datetime('now'))`, row.ID, userID, body)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the update.")
		return
	}
	id, _ := res.LastInsertId()
	h.notifyAttendees(row.ID, userID)
	update, err := h.updateByID(r, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the update.")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, update)
}

func (h *handler) commentByID(r *http.Request, id int64) (EventComment, error) {
	var c EventComment
	var authorID int64
	var parentID sql.NullInt64
	var body sql.NullString
	var deletedAt sql.NullString
	err := h.d.DB.QueryRowContext(r.Context(), `
		SELECT id, event_id, author_id, parent_comment_id, body, created_at, deleted_at
		FROM event_comments WHERE id = ?`, id).
		Scan(&c.ID, &c.EventID, &authorID, &parentID, &body, &c.CreatedAt, &deletedAt)
	if err != nil {
		return c, err
	}
	c.Deleted = deletedAt.Valid
	if c.Deleted {
		c.Body = ""
	} else {
		c.Body = body.String
	}
	if parentID.Valid {
		value := parentID.Int64
		c.ParentID = &value
	}
	users := loadUsers(r.Context(), h.d, []int64{authorID})
	c.Author = users[authorID]
	if c.Author.ID == 0 {
		c.Author = UserSummary{ID: authorID, DisplayName: "unknown"}
	}
	return c, nil
}

func (h *handler) listComments(w http.ResponseWriter, r *http.Request) {
	row, ok := h.subresourceEvent(w, r, viewerID(r, h.d))
	if !ok {
		return
	}
	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		var err error
		cursor, err = shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "Invalid cursor."})
			return
		}
	}
	rows, err := h.d.DB.QueryContext(r.Context(), `
		SELECT id FROM event_comments WHERE event_id = ? AND id < ? ORDER BY id DESC LIMIT ?`,
		row.ID, cursor, p.Limit+1)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list comments.")
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
	result := page[EventComment]{Items: []EventComment{}, NextCursor: ""}
	if len(ids) > p.Limit {
		ids = ids[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(ids[len(ids)-1])
	}
	for _, id := range ids {
		c, err := h.commentByID(r, id)
		if err == nil {
			result.Items = append(result.Items, c)
		}
	}
	shared.WriteJSON(w, http.StatusOK, result)
}

// commentDepth validates that the parent belongs to the event and that a new
// reply would not exceed the two-level nesting limit (depth 0..2).
func (h *handler) commentDepth(r *http.Request, eventID, parentID int64) error {
	depth := 0
	current := parentID
	for i := 0; i < 8; i++ {
		var next sql.NullInt64
		err := h.d.DB.QueryRowContext(r.Context(),
			`SELECT parent_comment_id FROM event_comments WHERE id = ? AND event_id = ?`,
			current, eventID).Scan(&next)
		if err != nil {
			return errors.New("invalid parent")
		}
		if !next.Valid {
			return nil
		}
		depth++
		if depth >= maxCommentDepth {
			return errors.New("too deep")
		}
		current = next.Int64
	}
	return errors.New("too deep")
}

// addComment returns the created EventComment JSON (201) for addEventComment.
func (h *handler) addComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	row, ok := h.subresourceEvent(w, r, userID)
	if !ok {
		return
	}
	if row.Status == "cancelled" {
		shared.WriteError(w, http.StatusConflict, codeEventCancelled, "This event has been cancelled.")
		return
	}
	var in struct {
		Body      string `json:"body"`
		ParentID  *int64 `json:"parentId"`
		ParentAlt *int64 `json:"parent_id"`
	}
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	body := strings.TrimSpace(in.Body)
	if body == "" || utf8.RuneCountInString(body) > maxBodyRunes {
		writeValidation(w, map[string]string{"body": "Comment must be between 1 and 2000 characters."})
		return
	}
	parentID := in.ParentID
	if parentID == nil {
		parentID = in.ParentAlt
	}
	var parent sql.NullInt64
	if parentID != nil {
		if *parentID <= 0 {
			writeValidation(w, map[string]string{"parentId": "Invalid parent comment."})
			return
		}
		if err := h.commentDepth(r, row.ID, *parentID); err != nil {
			writeValidation(w, map[string]string{"parentId": "Replies are limited to two levels."})
			return
		}
		parent = sql.NullInt64{Int64: *parentID, Valid: true}
	}
	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO event_comments(event_id, author_id, parent_comment_id, body, created_at)
		VALUES(?,?,?,?,datetime('now'))`, row.ID, userID, parent, body)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not publish the comment.")
		return
	}
	id, _ := res.LastInsertId()
	comment, err := h.commentByID(r, id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the comment.")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, comment)
}
