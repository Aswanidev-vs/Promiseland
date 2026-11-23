// Package creators implements creator discovery and curated featured posts
// for Release 2. Creators are derived from posts.creator_category and follows;
// no separate creator profile record is stored.
package creators

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeValidation = "validation_error"
	codeForbidden  = "forbidden"
	codeLimit      = "featured_limit"
	maxFeatured    = 6
)

// validatorCategory is the exact R2 creator-category vocabulary. The empty
// string means "not creator content" and is valid on a post but not as a
// discovery filter.
var validCategories = map[string]bool{
	"cosplay": true, "crossplay": true, "fan_art": true,
	"photography": true, "edit": true, "collection": true, "other": true,
}

// UserSummary is the exact R1 public user shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Creator is the exact R2 Creator wire shape. Featured posts use the shared
// feed/shape Post DTO so their JSON is identical to every other feed surface.
type Creator struct {
	User          UserSummary  `json:"user"`
	Categories    []string     `json:"categories"`
	FeaturedPosts []shape.Post `json:"featured_posts"`
	PostCount     int          `json:"post_count"`
	FollowerCount int          `json:"follower_count"`
}

// page always serialises next_cursor, including the empty end-of-list value.
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

type service struct {
	d      *shared.Deps
	schema *shape.Schema
	loader *shape.Loader
}

// Register mounts the public creator discovery routes.
func Register(r chi.Router, d *shared.Deps) {
	if r == nil {
		return
	}
	s := newService(d)
	r.Get("/creators", s.list)
	r.Get("/creators/{username}", s.getByUsername)
}

// RegisterProtected mounts the authenticated featured-post routes. The
// orchestrator passes a router already wrapped with shared.RequireAuth; the
// handler still verifies shared.UserID before touching the database.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	if r == nil {
		return
	}
	s := newService(d)
	r.Post("/users/me/featured/{postId}", s.feature)
	r.Delete("/users/me/featured/{postId}", s.unfeature)
}

func newService(d *shared.Deps) *service {
	var db *sql.DB
	if d != nil {
		db = d.DB
	}
	schema := shape.LoadSchema(context.Background(), db)
	return &service{
		d:      d,
		schema: schema,
		loader: &shape.Loader{DB: db, S: schema},
	}
}

func (s *service) writeInternal(w http.ResponseWriter, action string, err error) {
	if s != nil && s.d != nil && s.d.Log != nil {
		s.d.Log.Warn("creators: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

// viewerID accepts both public and protected routers. Protected routes carry
// the ID in context; public routes fall back to the session cookie so privacy
// and block filters still see signed-in members.
func viewerID(r *http.Request, d *shared.Deps) int64 {
	if id, ok := shared.UserID(r); ok {
		return id
	}
	if d == nil {
		return 0
	}
	id, _ := d.SessionUserID(r)
	return id
}

func normalizeCategory(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func nullString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// creatorAvailable returns the SQL predicate and bound arguments that hide
// private, non-discoverable, inactive and blocked users from the viewer.
func (s *service) creatorAvailable(viewerID int64) (string, []any) {
	var clauses []string
	var args []any
	sc := s.schema

	if sc.HasCol("users", "is_active") {
		clauses = append(clauses, "COALESCE(u.is_active, 1) = 1")
	}
	if sc.HasCol("user_profiles", "visibility") {
		clauses = append(clauses, "(COALESCE(up.visibility, 'public') = 'public' OR u.id = ?)")
		args = append(args, viewerID)
	}
	if sc.HasCol("users", "discoverable") {
		clauses = append(clauses, "(COALESCE(u.discoverable, 1) = 1 OR u.id = ?)")
		args = append(args, viewerID)
	}
	if sc.HasTable("suspensions") && sc.HasCol("suspensions", "user_id") {
		clauses = append(clauses, `NOT EXISTS (SELECT 1 FROM suspensions su
			WHERE su.user_id = u.id AND su.is_active = 1
			  AND (su.expires_at IS NULL OR su.expires_at > datetime('now')))`)
	}
	if sc.HasTable("blocks") {
		clauses = append(clauses, `NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE (b.blocker_id = u.id AND b.blocked_id = ?)
			   OR (b.blocker_id = ? AND b.blocked_id = u.id))`)
		args = append(args, viewerID, viewerID)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return strings.Join(clauses, " AND "), args
}
