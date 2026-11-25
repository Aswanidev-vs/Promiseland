// Package admin exposes the staff console API: monitoring counts, browsable
// tables and the account action the panel needs. Every route except /admin/me
// requires a platform staff grant.
package admin

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeForbidden  = "forbidden"
	codeValidation = "validation_error"
	queryTimeout   = 3 * time.Second
)

type module struct {
	d *shared.Deps
}

// Register mounts the console under a router that already requires a session.
func Register(r chi.Router, d *shared.Deps) {
	m := &module{d: d}

	// The panel asks this on load to decide whether to show itself; it answers
	// for any signed-in caller instead of 403 so the client can stay quiet.
	r.Get("/admin/me", m.me)

	staff := r.With(m.requireStaff)
	staff.Get("/admin/overview", m.overview)
	staff.Get("/admin/users", m.users)
	staff.Patch("/admin/users/{id}", m.updateUser)
	staff.Get("/admin/posts", m.posts)
	staff.Get("/admin/communities", m.communities)
	staff.Get("/admin/events", m.events)
}

type staffReply struct {
	IsStaff bool `json:"is_staff"`
}

func (m *module) me(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, staffReply{IsStaff: m.isStaff(r.Context(), userID)})
}

func (m *module) requireStaff(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := shared.UserID(r)
		if !ok {
			shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
			return
		}
		if !m.isStaff(r.Context(), userID) {
			shared.WriteError(w, http.StatusForbidden, codeForbidden, "Staff access is required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *module) isStaff(ctx context.Context, userID int64) bool {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
		return false
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var one int
	if err := m.d.DB.QueryRowContext(ctx,
		`SELECT 1 FROM platform_moderators WHERE user_id = ? LIMIT 1`, userID).Scan(&one); err == nil {
		return true
	}
	// Older databases carry the role as a boolean column.
	for _, column := range []string{"is_platform_moderator", "is_moderator"} {
		if !columnExists(m.d, "users", column) {
			continue
		}
		query := fmt.Sprintf(`SELECT COALESCE(%s, 0) FROM users WHERE id = ?`, column)
		if err := m.d.DB.QueryRowContext(ctx, query, userID).Scan(&one); err == nil && one != 0 {
			return true
		}
	}
	return false
}

func columnExists(d *shared.Deps, table, column string) bool {
	rows, err := d.DB.Query(`PRAGMA table_info("` + table + `")`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, kind string
		var notNull, pk int
		var dflt sql.NullString
		if rows.Scan(&cid, &name, &kind, &notNull, &dflt, &pk) != nil {
			return false
		}
		if name == column {
			return true
		}
	}
	return false
}

// ── Overview ────────────────────────────────────────────────────────────────

type counts struct {
	Users          int64 `json:"users"`
	ActiveUsers    int64 `json:"active_users"`
	SuspendedUsers int64 `json:"suspended_users"`
	ActiveSessions int64 `json:"active_sessions"`
	Posts          int64 `json:"posts"`
	Comments       int64 `json:"comments"`
	Communities    int64 `json:"communities"`
	Events         int64 `json:"events"`
	Media          int64 `json:"media"`
	MediaBytes     int64 `json:"media_bytes"`
	OpenReports    int64 `json:"open_reports"`
	PendingInvites int64 `json:"pending_memberships"`
}

type recentUser struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

type recentPost struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Community string `json:"community"`
	CreatedAt string `json:"created_at"`
}

type recentReport struct {
	ID         int64  `json:"id"`
	TargetType string `json:"target_type"`
	TargetID   int64  `json:"target_id"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"created_at"`
}

type overviewReply struct {
	Counts      counts         `json:"counts"`
	RecentUsers []recentUser   `json:"recent_users"`
	RecentPosts []recentPost   `json:"recent_posts"`
	OpenReports []recentReport `json:"open_reports"`
}

func (m *module) overview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	var c counts
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM users),
		  (SELECT COUNT(*) FROM users WHERE is_active = 1),
		  (SELECT COUNT(*) FROM users WHERE is_active = 0),
		  (SELECT COUNT(*) FROM sessions WHERE expires_at > datetime('now')),
		  (SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL),
		  (SELECT COUNT(*) FROM comments WHERE deleted_at IS NULL),
		  (SELECT COUNT(*) FROM communities),
		  (SELECT COUNT(*) FROM events),
		  (SELECT COUNT(*) FROM media),
		  (SELECT COALESCE(SUM(size_bytes), 0) FROM media),
		  (SELECT COUNT(*) FROM reports WHERE status = 'open'),
		  (SELECT COUNT(*) FROM community_memberships WHERE status = 'pending')`,
	).Scan(
		&c.Users, &c.ActiveUsers, &c.SuspendedUsers, &c.ActiveSessions,
		&c.Posts, &c.Comments, &c.Communities, &c.Events,
		&c.Media, &c.MediaBytes, &c.OpenReports, &c.PendingInvites,
	)
	if err != nil {
		m.fail(w, "overview counts", err)
		return
	}

	reply := overviewReply{
		Counts:      c,
		RecentUsers: []recentUser{},
		RecentPosts: []recentPost{},
		OpenReports: []recentReport{},
	}

	if rows, err := m.d.DB.QueryContext(ctx, `
		SELECT id, username, display_name, created_at
		FROM users ORDER BY id DESC LIMIT 5`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var item recentUser
			if rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.CreatedAt) == nil {
				reply.RecentUsers = append(reply.RecentUsers, item)
			}
		}
	}

	if rows, err := m.d.DB.QueryContext(ctx, `
		SELECT p.id, p.title, u.username, COALESCE(c.name, ''), p.created_at
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.deleted_at IS NULL
		ORDER BY p.id DESC LIMIT 5`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var item recentPost
			if rows.Scan(&item.ID, &item.Title, &item.Author, &item.Community, &item.CreatedAt) == nil {
				reply.RecentPosts = append(reply.RecentPosts, item)
			}
		}
	}

	if rows, err := m.d.DB.QueryContext(ctx, `
		SELECT id, target_type, target_id, reason, created_at
		FROM reports WHERE status = 'open'
		ORDER BY id DESC LIMIT 5`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var item recentReport
			if rows.Scan(&item.ID, &item.TargetType, &item.TargetID, &item.Reason, &item.CreatedAt) == nil {
				reply.OpenReports = append(reply.OpenReports, item)
			}
		}
	}

	shared.WriteJSON(w, http.StatusOK, reply)
}

// ── Users ───────────────────────────────────────────────────────────────────

type userRow struct {
	ID             int64  `json:"id"`
	Username       string `json:"username"`
	DisplayName    string `json:"display_name"`
	Email          string `json:"email"`
	IsActive       bool   `json:"is_active"`
	IsStaff        bool   `json:"is_staff"`
	Posts          int64  `json:"post_count"`
	ActiveSessions int64  `json:"active_sessions"`
	CreatedAt      string `json:"created_at"`
}

type usersReply struct {
	Items      []userRow `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

func (m *module) users(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		decoded, err := shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Invalid cursor.",
				map[string]string{"cursor": "Invalid cursor."})
			return
		}
		cursor = decoded
	}
	query := escapeLike(strings.TrimSpace(r.URL.Query().Get("q")))

	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT u.id, u.username, u.display_name, u.email, u.is_active,
		       EXISTS(SELECT 1 FROM platform_moderators pm WHERE pm.user_id = u.id),
		       (SELECT COUNT(*) FROM posts p WHERE p.author_id = u.id AND p.deleted_at IS NULL),
		       (SELECT COUNT(*) FROM sessions s WHERE s.user_id = u.id AND s.expires_at > datetime('now')),
		       u.created_at
		FROM users u
		WHERE u.id < ?
		  AND (? = '' OR lower(u.username) LIKE ? ESCAPE '\'
		                OR lower(u.display_name) LIKE ? ESCAPE '\'
		                OR lower(u.email) LIKE ? ESCAPE '\')
		ORDER BY u.id DESC
		LIMIT ?`, cursor, query, "%"+query+"%", "%"+query+"%", "%"+query+"%", p.Limit+1)
	if err != nil {
		m.fail(w, "list users", err)
		return
	}
	defer rows.Close()

	reply := usersReply{Items: []userRow{}}
	for rows.Next() {
		var item userRow
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Email,
			&item.IsActive, &item.IsStaff, &item.Posts, &item.ActiveSessions, &item.CreatedAt); err != nil {
			m.fail(w, "scan user", err)
			return
		}
		reply.Items = append(reply.Items, item)
	}
	if len(reply.Items) > p.Limit {
		reply.Items = reply.Items[:p.Limit]
		reply.NextCursor = shared.EncodeIDCursor(reply.Items[len(reply.Items)-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, reply)
}

type userPatch struct {
	IsActive *bool `json:"is_active"`
}

type userStateReply struct {
	ID       int64 `json:"id"`
	IsActive bool  `json:"is_active"`
}

func (m *module) updateUser(w http.ResponseWriter, r *http.Request) {
	viewerID, _ := shared.UserID(r)
	targetID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || targetID <= 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Invalid user id.",
			map[string]string{"id": "must be a positive integer"})
		return
	}
	var in userPatch
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	if in.IsActive == nil {
		shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Nothing to change.",
			map[string]string{"is_active": "required"})
		return
	}
	if targetID == viewerID && !*in.IsActive {
		shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "You cannot suspend your own account.",
			map[string]string{"is_active": "cannot disable the signed-in account"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	active := 0
	if *in.IsActive {
		active = 1
	}
	res, err := m.d.DB.ExecContext(ctx, `UPDATE users SET is_active = ? WHERE id = ?`, active, targetID)
	if err != nil {
		m.fail(w, "update user", err)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	// Suspension ends access now, not when the cookie expires.
	if active == 0 {
		if _, err := m.d.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, targetID); err != nil {
			m.fail(w, "revoke sessions", err)
			return
		}
	}
	shared.WriteJSON(w, http.StatusOK, userStateReply{ID: targetID, IsActive: active == 1})
}

// ── Posts ───────────────────────────────────────────────────────────────────

type postRow struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Author    string  `json:"author"`
	Community string  `json:"community"`
	Comments  int64   `json:"comment_count"`
	Score     int64   `json:"vote_score"`
	DeletedAt *string `json:"deleted_at"`
	CreatedAt string  `json:"created_at"`
}

type postsReply struct {
	Items      []postRow `json:"items"`
	NextCursor string    `json:"next_cursor"`
}

func (m *module) posts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		decoded, err := shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Invalid cursor.",
				map[string]string{"cursor": "Invalid cursor."})
			return
		}
		cursor = decoded
	}
	query := escapeLike(strings.TrimSpace(r.URL.Query().Get("q")))

	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT p.id, p.title, p.type, u.username, COALESCE(c.name, ''),
		       (SELECT COUNT(*) FROM comments cm WHERE cm.post_id = p.id AND cm.deleted_at IS NULL),
		       p.vote_score, p.deleted_at, p.created_at
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.id < ?
		  AND (? = '' OR lower(p.title) LIKE ? ESCAPE '\')
		ORDER BY p.id DESC
		LIMIT ?`, cursor, query, "%"+query+"%", p.Limit+1)
	if err != nil {
		m.fail(w, "list posts", err)
		return
	}
	defer rows.Close()

	reply := postsReply{Items: []postRow{}}
	for rows.Next() {
		var item postRow
		var deleted sql.NullString
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Author, &item.Community,
			&item.Comments, &item.Score, &deleted, &item.CreatedAt); err != nil {
			m.fail(w, "scan post", err)
			return
		}
		if deleted.Valid {
			item.DeletedAt = &deleted.String
		}
		reply.Items = append(reply.Items, item)
	}
	if len(reply.Items) > p.Limit {
		reply.Items = reply.Items[:p.Limit]
		reply.NextCursor = shared.EncodeIDCursor(reply.Items[len(reply.Items)-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, reply)
}

// ── Communities ─────────────────────────────────────────────────────────────

type communityRow struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Visibility string `json:"visibility"`
	Owner      string `json:"owner"`
	Members    int64  `json:"member_count"`
	Posts      int64  `json:"post_count"`
	CreatedAt  string `json:"created_at"`
}

type communitiesReply struct {
	Items      []communityRow `json:"items"`
	NextCursor string         `json:"next_cursor"`
}

func (m *module) communities(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		decoded, err := shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Invalid cursor.",
				map[string]string{"cursor": "Invalid cursor."})
			return
		}
		cursor = decoded
	}
	query := escapeLike(strings.TrimSpace(r.URL.Query().Get("q")))

	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT c.id, c.name, c.slug, c.visibility, u.username,
		       (SELECT COUNT(*) FROM community_memberships m WHERE m.community_id = c.id AND m.status = 'active'),
		       (SELECT COUNT(*) FROM posts p WHERE p.community_id = c.id AND p.deleted_at IS NULL),
		       c.created_at
		FROM communities c
		JOIN users u ON u.id = c.owner_id
		WHERE c.id < ?
		  AND (? = '' OR lower(c.name) LIKE ? ESCAPE '\' OR lower(c.slug) LIKE ? ESCAPE '\')
		ORDER BY c.id DESC
		LIMIT ?`, cursor, query, "%"+query+"%", "%"+query+"%", p.Limit+1)
	if err != nil {
		m.fail(w, "list communities", err)
		return
	}
	defer rows.Close()

	reply := communitiesReply{Items: []communityRow{}}
	for rows.Next() {
		var item communityRow
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Visibility, &item.Owner,
			&item.Members, &item.Posts, &item.CreatedAt); err != nil {
			m.fail(w, "scan community", err)
			return
		}
		reply.Items = append(reply.Items, item)
	}
	if len(reply.Items) > p.Limit {
		reply.Items = reply.Items[:p.Limit]
		reply.NextCursor = shared.EncodeIDCursor(reply.Items[len(reply.Items)-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, reply)
}

// ── Events ──────────────────────────────────────────────────────────────────

type eventRow struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"event_type"`
	Organizer string `json:"organizer"`
	City      string `json:"city"`
	StartsAt  string `json:"starts_at"`
	Status    string `json:"status"`
	Online    bool   `json:"is_online"`
	Going     int64  `json:"going_count"`
}

type eventsReply struct {
	Items      []eventRow `json:"items"`
	NextCursor string     `json:"next_cursor"`
}

func (m *module) events(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()

	p := shared.ParsePagination(r)
	cursor := int64(math.MaxInt64)
	if p.Cursor != "" {
		decoded, err := shared.DecodeIDCursor(p.Cursor)
		if err != nil {
			shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Invalid cursor.",
				map[string]string{"cursor": "Invalid cursor."})
			return
		}
		cursor = decoded
	}
	query := escapeLike(strings.TrimSpace(r.URL.Query().Get("q")))

	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT e.id, e.title, e.event_type, u.username, COALESCE(e.city, ''),
		       e.starts_at, e.status, e.is_online,
		       (SELECT COUNT(*) FROM event_rsvps r WHERE r.event_id = e.id AND r.status = 'going')
		FROM events e
		JOIN users u ON u.id = e.organizer_id
		WHERE e.id < ?
		  AND (? = '' OR lower(e.title) LIKE ? ESCAPE '\' OR lower(COALESCE(e.city, '')) LIKE ? ESCAPE '\')
		ORDER BY e.id DESC
		LIMIT ?`, cursor, query, "%"+query+"%", "%"+query+"%", p.Limit+1)
	if err != nil {
		m.fail(w, "list events", err)
		return
	}
	defer rows.Close()

	reply := eventsReply{Items: []eventRow{}}
	for rows.Next() {
		var item eventRow
		if err := rows.Scan(&item.ID, &item.Title, &item.Type, &item.Organizer, &item.City,
			&item.StartsAt, &item.Status, &item.Online, &item.Going); err != nil {
			m.fail(w, "scan event", err)
			return
		}
		reply.Items = append(reply.Items, item)
	}
	if len(reply.Items) > p.Limit {
		reply.Items = reply.Items[:p.Limit]
		reply.NextCursor = shared.EncodeIDCursor(reply.Items[len(reply.Items)-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, reply)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func (m *module) fail(w http.ResponseWriter, action string, err error) {
	if m.d.Log != nil {
		m.d.Log.Error("admin: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"Could not complete that request.")
}

// escapeLike keeps % and _ literal inside the pattern.
func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}
