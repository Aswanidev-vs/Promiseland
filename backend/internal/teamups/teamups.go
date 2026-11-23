// Package teamups implements Release 2 collaboration / Team Up requests:
// creation, discovery, applications and owner-only review.
package teamups

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
	codeApplied    = "already_applied"
	codeSelfApply  = "cannot_apply_to_own_team_up"
)

// allowedCategories is the union of R2 creator categories and event-related
// tags. Team-ups must carry between one and five of these tokens.
var allowedCategories = map[string]bool{
	"cosplay": true, "crossplay": true, "fan_art": true,
	"photography": true, "edit": true, "collection": true, "other": true,
	"convention": true, "meetup": true, "screening": true, "photoshoot": true,
	"club": true, "online": true, "competition": true,
}

var validStatuses = map[string]bool{
	"open": true, "filled": true, "closed": true,
}

// UserSummary is the exact R1 public user shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Event mirrors the R2 Event wire shape. It is loaded defensively from the
// events table owned by the parallel events agent; a missing/incomplete table
// simply yields a null event on the TeamUp JSON.
type Event struct {
	ID          int64       `json:"id"`
	Organizer   UserSummary `json:"organizer"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	EventType   string      `json:"event_type"`
	StartsAt    string      `json:"starts_at"`
	EndsAt      string      `json:"ends_at"`
	Timezone    string      `json:"timezone"`
	VenueText   string      `json:"venue_text"`
	City        string      `json:"city"`
	Country     string      `json:"country"`
	IsOnline    bool        `json:"is_online"`
	Capacity    int         `json:"capacity"`
	RSVPCount   int         `json:"rsvp_count"`
	RSVPStatus  *string     `json:"rsvp_status"`
	CoverURL    *string     `json:"cover_url"`
	Status      string      `json:"status"`
	CreatedAt   string      `json:"created_at"`
}

// TeamUp is the exact R2 TeamUp wire shape.
type TeamUp struct {
	ID             int64       `json:"id"`
	Owner          UserSummary `json:"owner"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Event          *Event      `json:"event"`
	Categories     []string    `json:"categories"`
	Status         string      `json:"status"`
	ApplicantCount int         `json:"applicant_count"`
	MyApplication  *string     `json:"my_application"`
	CreatedAt      string      `json:"created_at"`
}

// TeamUpApplication is the exact R2 TeamUpApplication wire shape.
type TeamUpApplication struct {
	ID        int64       `json:"id"`
	TeamUpID  int64       `json:"team_up_id"`
	Applicant UserSummary `json:"applicant"`
	Message   string      `json:"message"`
	Status    string      `json:"status"`
	CreatedAt string      `json:"created_at"`
}

// page always serialises next_cursor, including the empty end-of-list value.
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

type service struct {
	d      *shared.Deps
	schema *shape.Schema
}

// Register mounts the public team-up read routes.
func Register(r chi.Router, d *shared.Deps) {
	if r == nil {
		return
	}
	s := newService(d)
	r.Get("/team-ups", s.list)
	r.Get("/team-ups/{id}", s.get)
}

// RegisterProtected mounts authenticated team-up routes. The orchestrator
// passes a router already wrapped with shared.RequireAuth; every handler still
// checks shared.UserID and resource ownership itself.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	if r == nil {
		return
	}
	s := newService(d)
	r.Post("/team-ups", s.create)
	r.Patch("/team-ups/{id}", s.patch)
	r.Delete("/team-ups/{id}", s.remove)
	r.Post("/team-ups/{id}/apply", s.apply)
	r.Get("/team-ups/{id}/applications", s.listApplications)
	r.Patch("/team-ups/{id}/applications/{applicationId}", s.reviewApplication)
}

func newService(d *shared.Deps) *service {
	var db *sql.DB
	if d != nil {
		db = d.DB
	}
	return &service{d: d, schema: shape.LoadSchema(context.Background(), db)}
}

func (s *service) writeInternal(w http.ResponseWriter, action string, err error) {
	if s != nil && s.d != nil && s.d.Log != nil {
		s.d.Log.Warn("teamups: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

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

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func nullString(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	s := v.String
	return &s
}

// ownerAvailable hides inactive, private, non-discoverable and blocked owners
// from discovery. The fragment uses only trusted, compile-time SQL; viewer IDs
// are always bound through arguments.
func (s *service) ownerAvailable(viewerID int64) (string, []any) {
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

// teamUpExists returns the owner ID for a team-up or sql.ErrNoRows.
func (s *service) teamUpExists(ctx context.Context, id int64) (int64, error) {
	var ownerID int64
	err := s.d.DB.QueryRowContext(ctx,
		`SELECT owner_id FROM team_ups WHERE id = ?`, id).Scan(&ownerID)
	return ownerID, err
}
