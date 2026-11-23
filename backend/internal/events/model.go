// Package events implements event discovery, RSVPs, organiser updates and
// event discussion for Release 2.
package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // IANA zone validation must also work on Windows hosts.

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeForbidden       = "forbidden"
	codeValidation      = "validation_error"
	codeEventFull       = "event_full"
	codeEventCancelled  = "event_cancelled"
	codeRSVPRequired    = "rsvp_required"
	codeInvalidCursor   = "invalid_cursor"
	maxTitleRunes       = 120
	maxDescriptionRunes = 5000
	maxBodyRunes        = 2000
	maxPlaceRunes       = 80
	maxVenueRunes       = 2000
	maxCommentDepth     = 2
)

// allowedTypes is the R2 event type vocabulary.
var allowedTypes = map[string]bool{
	"convention": true, "meetup": true, "screening": true,
	"photoshoot": true, "club": true, "online": true,
	"competition": true, "other": true,
}

var allowedStatuses = map[string]bool{
	"scheduled": true, "cancelled": true, "completed": true,
}

// UserSummary is the exact shared public user shape.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// Event is the exact R2 event shape. RSVPCount counts "going" RSVPs only;
// "maybe" RSVPs do not consume capacity. Capacity 0 means no limit.
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

// EventUpdate is the exact R2 update shape.
type EventUpdate struct {
	ID        int64       `json:"id"`
	EventID   int64       `json:"event_id"`
	Author    UserSummary `json:"author"`
	Body      string      `json:"body"`
	CreatedAt string      `json:"created_at"`
}

// EventComment is the exact R2 comment shape.
type EventComment struct {
	ID        int64       `json:"id"`
	EventID   int64       `json:"event_id"`
	Author    UserSummary `json:"author"`
	ParentID  *int64      `json:"parent_id"`
	Body      string      `json:"body"`
	CreatedAt string      `json:"created_at"`
	Deleted   bool        `json:"deleted"`
}

// page always serialises next_cursor, including when it is empty.
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

type eventRow struct {
	Event
	OrganizerID  int64
	CoverMediaID sql.NullInt64
}

// eventFacts holds the columns needed for ownership/capacity checks.
type eventFacts struct {
	ID          int64
	OrganizerID int64
	Status      string
	Capacity    int
	StartsAt    string
	EndsAt      string
}

// optionalString distinguishes an absent JSON field from an explicit null so
// patch handlers can clear nullable columns.
type optionalString struct {
	Set   bool
	Value *string
}

func (o *optionalString) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

type optionalInt64 struct {
	Set   bool
	Value *int64
}

func (o *optionalInt64) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	o.Value = &n
	return nil
}

type handler struct{ d *shared.Deps }

// Register mounts public event routes. Authenticated visitors are detected via
// d.SessionUserID so cancelled-event visibility and viewer RSVP status still
// work when the orchestrator mounts this router without OptionalAuth.
func Register(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Get("/events", h.list)
	r.Get("/events/{id}", h.get)
	r.Get("/events/{id}/updates", h.listUpdates)
	r.Get("/events/{id}/comments", h.listComments)
}

// RegisterProtected mounts authenticated event routes. The orchestrator passes
// a chi router wrapped with shared.RequireAuth; methods are disjoint from the
// public registration so calling both never collides.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	h := &handler{d: d}
	r.Post("/events", h.create)
	r.Patch("/events/{id}", h.update)
	r.Delete("/events/{id}", h.cancel)
	r.Post("/events/{id}/rsvp", h.rsvp)
	r.Delete("/events/{id}/rsvp", h.cancelRSVP)
	r.Post("/events/{id}/updates", h.addUpdate)
	r.Post("/events/{id}/comments", h.addComment)
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Please correct the highlighted fields.", details)
}

// viewerID accepts both routers: protected routes carry the ID in context,
// public routes fall back to the session cookie.
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

// requireUser is a defensive check in addition to the router's RequireAuth:
// handlers must never trust an unwrapped protected router.
func requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, ok := shared.UserID(r)
	if !ok || id <= 0 {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return 0, false
	}
	return id, true
}

func tableExists(d *shared.Deps, name string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	var n int
	err := d.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, name,
	).Scan(&n)
	return err == nil && n > 0
}

func columnExists(d *shared.Deps, table, column string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	var n int
	err := d.DB.QueryRow(fmt.Sprintf(
		`SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?`, table,
	), column).Scan(&n)
	return err == nil && n > 0
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

// escapeLike protects LIKE filters from user-supplied wildcards.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizeTimezone(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "UTC"
	}
	if _, err := time.LoadLocation(raw); err != nil {
		return "UTC"
	}
	return raw
}

func parseEventTime(raw string) (string, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return "", err
	}
	return t.UTC().Format(time.RFC3339), nil
}

// parseFilterTime accepts RFC3339 timestamps and bare calendar dates.
func parseFilterTime(raw string, inclusiveEnd bool) (string, error) {
	s := strings.TrimSpace(raw)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if inclusiveEnd {
			t = t.AddDate(0, 0, 1).Add(-time.Second)
		}
		return t.UTC().Format(time.RFC3339), nil
	}
	return parseEventTime(s)
}

func parseCursor(encoded string) (string, int64, error) {
	raw, err := shared.DecodeCursor(encoded)
	if err != nil {
		return "", 0, err
	}
	parts := strings.Split(raw, "|")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid cursor")
	}
	if _, err := time.Parse(time.RFC3339, parts[0]); err != nil {
		return "", 0, fmt.Errorf("invalid cursor")
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id < 0 {
		return "", 0, fmt.Errorf("invalid cursor")
	}
	return parts[0], id, nil
}

func encodeEventCursor(startsAt string, id int64) string {
	return shared.EncodeCursor(startsAt + "|" + strconv.FormatInt(id, 10))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
