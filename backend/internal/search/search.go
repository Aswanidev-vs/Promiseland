// Package search provides the R1 public search endpoint. It never returns
// private communities, private profiles, blocked users or deleted posts.
package search

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	errValidation = "validation_error"
	maxQueryRunes = 100
)

type module struct {
	d      *shared.Deps
	s      *shape.Schema
	loader *shape.Loader
}

// Register mounts the public search route on r.
func Register(r chi.Router, d *shared.Deps) {
	m := newModule(context.Background(), d)
	if r == nil {
		return
	}
	r.Get("/search", m.handleSearch)
}

// RegisterProtected is the authenticated variant; search works the same for
// both audiences, with signed-in callers seeing only what they are allowed to.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	Register(r, d)
}

func newModule(ctx context.Context, d *shared.Deps) *module {
	var db *sql.DB
	if d != nil {
		db = d.DB
	}
	s := shape.LoadSchema(ctx, db)
	return &module{d: d, s: s, loader: &shape.Loader{DB: db, S: s}}
}

func (m *module) handleSearch(w http.ResponseWriter, r *http.Request) {
	if m == nil || m.d == nil || m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Search is temporarily unavailable.")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	runes := utf8.RuneCountInString(q)
	if runes < 1 || runes > maxQueryRunes {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Search query must be between 1 and 100 characters.",
			map[string]string{"q": "length must be between 1 and 100"})
		return
	}
	typeFilter := strings.TrimSpace(r.URL.Query().Get("type"))
	if typeFilter == "" {
		typeFilter = "all"
	}
	switch typeFilter {
	case "all", "users", "communities", "posts":
	default:
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid search type.", map[string]string{"type": "must be users, communities, posts or all"})
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

	viewerID, _ := m.d.SessionUserID(r)
	pattern := "%" + shape.EscapeLike(q) + "%"
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	results := struct {
		Users       []shape.UserSummary `json:"users"`
		Communities []shape.Community   `json:"communities"`
		Posts       []shape.Post        `json:"posts"`
	}{
		Users:       []shape.UserSummary{},
		Communities: []shape.Community{},
		Posts:       []shape.Post{},
	}

	if typeFilter == "all" || typeFilter == "users" {
		users, err := m.searchUsers(ctx, pattern, viewerID, pg.Limit, afterID)
		if err != nil {
			m.logError("user search failed", err)
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Search could not be completed.")
			return
		}
		results.Users = users
	}
	if typeFilter == "all" || typeFilter == "communities" {
		communities, err := m.searchCommunities(ctx, pattern, viewerID, pg.Limit, afterID)
		if err != nil {
			m.logError("community search failed", err)
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Search could not be completed.")
			return
		}
		results.Communities = communities
	}
	if typeFilter == "all" || typeFilter == "posts" {
		posts, err := m.searchPosts(ctx, pattern, viewerID, pg.Limit, afterID)
		if err != nil {
			m.logError("post search failed", err)
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Search could not be completed.")
			return
		}
		results.Posts = posts
	}

	shared.WriteJSON(w, http.StatusOK, results)
}

func (m *module) logError(message string, err error) {
	if m != nil && m.d != nil && m.d.Log != nil {
		m.d.Log.Warn(message, "error", err)
	}
}

func (m *module) searchUsers(ctx context.Context, pattern string, viewerID int64, limit int, afterID int64) ([]shape.UserSummary, error) {
	s := m.s
	if s == nil || !s.HasTable("users") {
		return []shape.UserSummary{}, nil
	}
	username := "''"
	switch {
	case s.HasCol("users", "username"):
		username = "COALESCE(NULLIF(u.username, ''), u.display_name, '')"
	case s.HasCol("user_profiles", "username"):
		username = "COALESCE(up.username, '')"
	}
	displayName := s.Text("users", "u", "display_name")
	if !s.HasCol("users", "display_name") {
		displayName = s.Text("user_profiles", "up", "display_name")
	}
	avatar := "CAST(NULL AS TEXT)"
	if s.HasCol("user_profiles", "avatar_url") {
		avatar = "up.avatar_url"
	} else if s.HasCol("users", "avatar_url") {
		avatar = "u.avatar_url"
	}

	var clauses []string
	var args []any
	clauses = append(clauses, fmt.Sprintf("(%s LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\')", username, displayName))
	args = append(args, pattern, pattern)

	if s.HasTable("user_profiles") && s.HasCol("user_profiles", "visibility") {
		clauses = append(clauses, fmt.Sprintf("(COALESCE(up.visibility, 'public') = 'public' OR u.id = %d)", viewerID))
	}
	if s.HasCol("users", "discoverable") {
		clauses = append(clauses, fmt.Sprintf("(COALESCE(u.discoverable, 1) = 1 OR u.id = %d)", viewerID))
	} else if s.HasTable("user_profiles") && s.HasCol("user_profiles", "discoverable") {
		clauses = append(clauses, fmt.Sprintf("(COALESCE(up.discoverable, 1) = 1 OR u.id = %d)", viewerID))
	}
	if s.HasCol("users", "is_active") {
		clauses = append(clauses, "COALESCE(u.is_active, 1) = 1")
	}
	if s.HasTable("blocks") {
		blocker := s.Col("blocks", "blocker_id", "user_id", "blocking_id")
		blocked := s.Col("blocks", "blocked_id", "blocking_user_id", "target_id")
		if blocker != "" && blocked != "" {
			clauses = append(clauses, fmt.Sprintf(`NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE (b.%s = ? AND b.%s = u.id) OR (b.%s = u.id AND b.%s = ?)
			)`, blocker, blocked, blocker, blocked))
			args = append(args, viewerID, viewerID)
		}
	}
	if s.HasTable("suspensions") {
		clauses = append(clauses, `NOT EXISTS (
			SELECT 1 FROM suspensions su
			WHERE su.user_id = u.id AND su.is_active = 1
			  AND (su.expires_at IS NULL OR su.expires_at > datetime('now'))
		)`)
	}
	if afterID > 0 {
		clauses = append(clauses, "u.id < ?")
		args = append(args, afterID)
	}

	q := fmt.Sprintf(`SELECT u.id, %s, %s, %s
		FROM users u LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE %s ORDER BY u.id DESC LIMIT %d`,
		username, displayName, avatar, strings.Join(clauses, " AND "), limit)
	rows, err := m.d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	out := []shape.UserSummary{}
	for rows.Next() {
		var user shape.UserSummary
		var avatarValue sql.NullString
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &avatarValue); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if avatarValue.Valid {
			v := avatarValue.String
			user.AvatarURL = &v
		}
		out = append(out, user)
	}
	return out, rows.Err()
}

func (m *module) searchCommunities(ctx context.Context, pattern string, viewerID int64, limit int, afterID int64) ([]shape.Community, error) {
	s := m.s
	if s == nil || !s.HasTable("communities") {
		return []shape.Community{}, nil
	}
	desc := "''"
	if s.HasCol("communities", "description") {
		desc = "COALESCE(c.description, '')"
	}
	clause := fmt.Sprintf(" AND (c.name LIKE ? ESCAPE '\\' OR %s LIKE ? ESCAPE '\\' OR c.slug LIKE ? ESCAPE '\\')", desc)
	args := []any{pattern, pattern, pattern}
	visibility, visibilityArgs := shape.CommunityVisibilityWhere(s, viewerID)
	clause += visibility
	args = append(args, visibilityArgs...)
	if afterID > 0 {
		clause += " AND c.id < ?"
		args = append(args, afterID)
	}
	return shape.LoadCommunities(ctx, m.d.DB, s, clause, args, viewerID, limit)
}

func (m *module) searchPosts(ctx context.Context, pattern string, viewerID int64, limit int, afterID int64) ([]shape.Post, error) {
	s := m.s
	tagCol := s.Col("post_tags", "tag", "tag_name", "name")
	clause := " AND (p.title LIKE ? ESCAPE '\\' OR p.body LIKE ? ESCAPE '\\'"
	args := []any{pattern, pattern}
	if tagCol != "" {
		clause += fmt.Sprintf(` OR EXISTS (
			SELECT 1 FROM post_tags pt WHERE pt.post_id = p.id AND pt.%s LIKE ? ESCAPE '\'
		)`, tagCol)
		args = append(args, pattern)
	}
	clause += ")"
	visibility, visibilityArgs := shape.VisibilityClauses(s, viewerID)
	if visibility != "" {
		clause += " AND " + visibility
		args = append(args, visibilityArgs...)
	}
	if afterID > 0 {
		clause += " AND p.id < ?"
		args = append(args, afterID)
	}
	return m.loader.Load(ctx, clause, args, viewerID, limit)
}
