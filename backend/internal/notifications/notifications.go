// Package notifications serves the R1 in-app notification inbox and the
// per-category preference settings. Every handler resolves the viewer from
// the session cookie (or the auth middleware context) and enforces ownership.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	errValidation = "validation_error"
	errNotFound   = "not_found"
)

// Categories is the fixed set of R1 notification preference categories.
var Categories = []string{"follow", "reply", "mention", "reaction", "community", "event"}

type module struct {
	d *shared.Deps
	s *shape.Schema
}

// Register mounts notification routes on a public router. Handlers enforce
// authentication themselves, so this is safe even without RequireAuth.
func Register(r chi.Router, d *shared.Deps) {
	m := newModule(context.Background(), d)
	m.mount(r)
}

// RegisterProtected mounts the same routes on an authenticated router.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	Register(r, d)
}

func newModule(ctx context.Context, d *shared.Deps) *module {
	var db *sql.DB
	if d != nil {
		db = d.DB
	}
	return &module{d: d, s: shape.LoadSchema(ctx, db)}
}

func (m *module) mount(r chi.Router) {
	if r == nil {
		return
	}
	r.Get("/notifications", m.handleList)
	r.Post("/notifications/read-all", m.handleReadAll)
	r.Patch("/notifications/read-all", m.handleReadAll)
	r.Get("/notifications/preferences", m.handleGetPreferences)
	r.Put("/notifications/preferences", m.handlePutPreferences)
	r.Post("/notifications/{id}/read", m.handleMarkRead)
	r.Patch("/notifications/{id}/read", m.handleMarkRead)
}

func (m *module) viewer(r *http.Request) (int64, bool) {
	if id, ok := shared.UserID(r); ok {
		return id, true
	}
	if m == nil || m.d == nil {
		return 0, false
	}
	return m.d.SessionUserID(r)
}

func requireViewer(w http.ResponseWriter, r *http.Request, m *module) (int64, bool) {
	id, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return 0, false
	}
	return id, true
}

// Notification is the exact R1 Notification wire shape.
type Notification struct {
	ID         int64              `json:"id"`
	Type       string             `json:"type"`
	Actor      *shape.UserSummary `json:"actor"`
	TargetType string             `json:"target_type"`
	TargetID   int64              `json:"target_id"`
	ReadAt     *string            `json:"read_at"`
	CreatedAt  string             `json:"created_at"`
	Data       json.RawMessage    `json:"data"`
}

func (m *module) handleList(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	if m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Notifications are temporarily unavailable.")
		return
	}
	pg := shared.ParsePagination(r)
	var afterID int64
	if pg.Cursor != "" {
		id, err := shared.DecodeIDCursor(pg.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
				"Invalid pagination cursor.", map[string]string{"cursor": "must be a valid cursor"})
			return
		}
		afterID = id
	}

	items, err := m.list(r.Context(), viewerID, pg.Limit+1, afterID)
	if err != nil {
		m.logError("notification list failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Notifications could not be loaded.")
		return
	}
	page := shared.NewPage(items)
	if len(items) > pg.Limit {
		page.Items = items[:pg.Limit]
		page.NextCursor = shared.EncodeIDCursor(items[pg.Limit-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, page)
}

func (m *module) list(ctx context.Context, viewerID int64, limit int, afterID int64) ([]Notification, error) {
	if m == nil || m.s == nil || !m.s.HasTable("notifications") {
		return []Notification{}, nil
	}
	s := m.s
	actorExpr := "CAST(NULL AS INTEGER)"
	if s.HasCol("notifications", "actor_id") {
		actorExpr = "n.actor_id"
	}
	username := "''"
	switch {
	case s.HasCol("users", "username"):
		username = "COALESCE(NULLIF(u.username, ''), u.display_name, '')"
	case s.HasCol("user_profiles", "username"):
		username = "COALESCE(up.username, '')"
	}
	displayName := s.NullableText("users", "u", "display_name")
	if !s.HasCol("users", "display_name") {
		displayName = s.NullableText("user_profiles", "up", "display_name")
	}
	avatar := "CAST(NULL AS TEXT)"
	if s.HasCol("user_profiles", "avatar_url") {
		avatar = "up.avatar_url"
	} else if s.HasCol("users", "avatar_url") {
		avatar = "u.avatar_url"
	}
	dataExpr := "'{}'"
	if s.HasCol("notifications", "data") {
		dataExpr = "COALESCE(n.data, '{}')"
	}
	readAt := "CAST(NULL AS TEXT)"
	if s.HasCol("notifications", "read_at") {
		readAt = "n.read_at"
	}
	targetType := "''"
	if s.HasCol("notifications", "target_type") {
		targetType = "CAST(n.target_type AS TEXT)"
	}
	targetID := "0"
	if s.HasCol("notifications", "target_id") {
		targetID = "COALESCE(n.target_id, 0)"
	}
	createdAt := "''"
	if s.HasCol("notifications", "created_at") {
		createdAt = "CAST(n.created_at AS TEXT)"
	}

	clauses := []string{"n.user_id = ?"}
	args := []any{viewerID}
	if afterID > 0 {
		clauses = append(clauses, "n.id < ?")
		args = append(args, afterID)
	}
	q := fmt.Sprintf(`SELECT n.id, %s, u.id, %s, %s, %s,
		%s, %s, %s, %s, %s
		FROM notifications n
		LEFT JOIN users u ON u.id = %s
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE %s ORDER BY n.id DESC LIMIT %d`,
		s.Text("notifications", "n", "type"),
		username, displayName, avatar,
		targetType, targetID, readAt, createdAt, dataExpr,
		actorExpr, strings.Join(clauses, " AND "), limit,
	)
	rows, err := m.d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	out := []Notification{}
	for rows.Next() {
		var n Notification
		var nType, targetTypeValue, created string
		var targetIDValue int64
		var readAtValue sql.NullString
		var dataValue string
		var actorID sql.NullInt64
		var actorUsername, actorDisplayName string
		var actorAvatar sql.NullString
		if err := rows.Scan(&n.ID, &nType, &actorID, &actorUsername, &actorDisplayName, &actorAvatar,
			&targetTypeValue, &targetIDValue, &readAtValue, &created, &dataValue); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		n.Type = nType
		n.TargetType = targetTypeValue
		n.TargetID = targetIDValue
		n.CreatedAt = created
		if readAtValue.Valid {
			v := readAtValue.String
			n.ReadAt = &v
		}
		n.Data = validJSONObject(dataValue)
		if actorID.Valid {
			actor := &shape.UserSummary{ID: actorID.Int64, Username: actorUsername, DisplayName: actorDisplayName}
			if actorAvatar.Valid {
				v := actorAvatar.String
				actor.AvatarURL = &v
			}
			n.Actor = actor
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func validJSONObject(raw string) json.RawMessage {
	if raw == "" {
		return json.RawMessage(`{}`)
	}
	var probe any
	if json.Unmarshal([]byte(raw), &probe) != nil {
		return json.RawMessage(`{}`)
	}
	return json.RawMessage(raw)
}

func (m *module) handleMarkRead(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid notification id.", map[string]string{"id": "must be a positive integer"})
		return
	}
	if !m.s.HasTable("notifications") {
		shared.WriteError(w, http.StatusNotFound, errNotFound, "Notification not found.")
		return
	}
	query := "UPDATE notifications SET read_at = datetime('now') WHERE id = ? AND user_id = ?"
	if m.s.HasCol("notifications", "read_at") {
		query += " AND read_at IS NULL"
	}
	res, err := m.d.DB.ExecContext(r.Context(), query, id, viewerID)
	if err != nil {
		m.logError("mark notification read failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Notification could not be updated.")
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		var exists int
		err := m.d.DB.QueryRowContext(r.Context(),
			"SELECT 1 FROM notifications WHERE id = ? AND user_id = ? LIMIT 1", id, viewerID).Scan(&exists)
		if err != nil {
			shared.WriteError(w, http.StatusNotFound, errNotFound, "Notification not found.")
			return
		}
		// Already read: the operation is idempotent.
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *module) handleReadAll(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	if !m.s.HasTable("notifications") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	query := "UPDATE notifications SET read_at = datetime('now') WHERE user_id = ?"
	if m.s.HasCol("notifications", "read_at") {
		query += " AND read_at IS NULL"
	}
	if _, err := m.d.DB.ExecContext(r.Context(), query, viewerID); err != nil {
		m.logError("mark all notifications read failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Notifications could not be updated.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *module) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	prefs, err := m.preferences(r.Context(), viewerID)
	if err != nil {
		m.logError("load notification preferences failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Preferences could not be loaded.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, prefs)
}

func (m *module) handlePutPreferences(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	var body map[string]bool
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	if len(body) == 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"No notification categories supplied.", map[string]string{"categories": "at least one category is required"})
		return
	}
	updates := map[string]bool{}
	for key, enabled := range body {
		if !validCategory(key) {
			shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
				"Unknown notification category.", map[string]string{key: "unknown category"})
			return
		}
		updates[key] = enabled
	}
	if m.s.HasTable("notification_preferences") {
		for _, category := range Categories {
			enabled, present := updates[category]
			if !present {
				continue
			}
			value := 0
			if enabled {
				value = 1
			}
			_, err := m.d.DB.ExecContext(r.Context(), `
				INSERT INTO notification_preferences (user_id, category, enabled)
				VALUES (?, ?, ?)
				ON CONFLICT(user_id, category) DO UPDATE SET enabled = excluded.enabled`,
				viewerID, category, value)
			if err != nil {
				m.logError("save notification preferences failed", err)
				shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Preferences could not be saved.")
				return
			}
		}
	}
	prefs, err := m.preferences(r.Context(), viewerID)
	if err != nil {
		m.logError("reload notification preferences failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Preferences could not be saved.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, prefs)
}

func validCategory(category string) bool {
	for _, c := range Categories {
		if c == category {
			return true
		}
	}
	return false
}

func (m *module) preferences(ctx context.Context, viewerID int64) (map[string]bool, error) {
	prefs := map[string]bool{}
	for _, category := range Categories {
		prefs[category] = true
	}
	if m == nil || m.s == nil || !m.s.HasTable("notification_preferences") || m.d.DB == nil {
		return prefs, nil
	}
	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT category, enabled FROM notification_preferences WHERE user_id = ?`, viewerID)
	if err != nil {
		return nil, fmt.Errorf("load preferences: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var category string
		var enabled int
		if rows.Scan(&category, &enabled) == nil && validCategory(category) {
			prefs[category] = enabled != 0
		}
	}
	return prefs, rows.Err()
}

func (m *module) logError(message string, err error) {
	if m != nil && m.d != nil && m.d.Log != nil {
		m.d.Log.Warn(message, "error", err)
	}
}
