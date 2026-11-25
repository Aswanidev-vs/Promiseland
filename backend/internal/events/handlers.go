package events

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

// eventInput accepts both the conventions' snake_case keys and the camelCase
// keys used by the R2 frontend API client.
type eventInput struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	EventType       string `json:"event_type"`
	EventTypeAlt    string `json:"eventType"`
	StartsAt        string `json:"starts_at"`
	StartsAtAlt     string `json:"startsAt"`
	EndsAt          string `json:"ends_at"`
	EndsAtAlt       string `json:"endsAt"`
	Timezone        string `json:"timezone"`
	VenueText       string `json:"venue_text"`
	VenueTextAlt    string `json:"venueText"`
	City            string `json:"city"`
	Country         string `json:"country"`
	IsOnline        *bool  `json:"is_online"`
	IsOnlineAlt     *bool  `json:"isOnline"`
	Capacity        *int   `json:"capacity"`
	CoverMediaID    *int64 `json:"cover_media_id"`
	CoverMediaIDAlt *int64 `json:"coverMediaId"`
}

type eventValues struct {
	Title        string
	Description  string
	EventType    string
	StartsAt     string
	EndsAt       string
	Timezone     string
	VenueText    string
	City         string
	Country      string
	IsOnline     bool
	Capacity     int
	CoverMediaID *int64
}

func (in eventInput) normalize() eventValues {
	v := eventValues{
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		EventType:   strings.TrimSpace(firstNonEmpty(in.EventType, in.EventTypeAlt)),
		StartsAt:    strings.TrimSpace(firstNonEmpty(in.StartsAt, in.StartsAtAlt)),
		EndsAt:      strings.TrimSpace(firstNonEmpty(in.EndsAt, in.EndsAtAlt)),
		Timezone:    normalizeTimezone(in.Timezone),
		VenueText:   strings.TrimSpace(firstNonEmpty(in.VenueText, in.VenueTextAlt)),
		City:        strings.TrimSpace(in.City),
		Country:     strings.TrimSpace(in.Country),
	}
	if in.IsOnline != nil {
		v.IsOnline = *in.IsOnline
	} else if in.IsOnlineAlt != nil {
		v.IsOnline = *in.IsOnlineAlt
	}
	if in.Capacity != nil {
		v.Capacity = *in.Capacity
	}
	v.CoverMediaID = in.CoverMediaID
	if v.CoverMediaID == nil && in.CoverMediaIDAlt != nil && *in.CoverMediaIDAlt > 0 {
		v.CoverMediaID = in.CoverMediaIDAlt
	}
	if v.CoverMediaID != nil && *v.CoverMediaID <= 0 {
		v.CoverMediaID = nil
	}
	return v
}

func titleError(title string) string {
	n := utf8.RuneCountInString(title)
	if n < 3 || n > maxTitleRunes {
		return "Title must be between 3 and 120 characters."
	}
	return ""
}

func descriptionError(body string) string {
	if utf8.RuneCountInString(body) > maxDescriptionRunes {
		return "Description must be at most 5000 characters."
	}
	return ""
}

func typeError(typ string) string {
	if !allowedTypes[typ] {
		return "Choose a valid event type."
	}
	return ""
}

// timePairError validates two RFC3339 timestamps and their order.
func timePairError(starts, ends string) string {
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(starts))
	if err != nil {
		return "Start and end times must be RFC3339 timestamps."
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(ends))
	if err != nil {
		return "Start and end times must be RFC3339 timestamps."
	}
	if !end.After(start) {
		return "The end time must be after the start time."
	}
	return ""
}

func placeError(city, country, venue string) string {
	if utf8.RuneCountInString(city) > maxPlaceRunes || utf8.RuneCountInString(country) > maxPlaceRunes {
		return "City and country must be at most 80 characters."
	}
	if utf8.RuneCountInString(venue) > maxVenueRunes {
		return "Venue details must be at most 2000 characters."
	}
	return ""
}

func capacityError(capacity int) string {
	if capacity < 0 {
		return "Capacity cannot be negative."
	}
	return ""
}

func addIssue(details map[string]string, key, message string) {
	if message != "" && details[key] == "" {
		details[key] = message
	}
}

// validCover checks an optional cover media reference against the media table.
// A missing table is tolerated so events keep working if media is not migrated.
// validCover accepts an absent cover, or media owned by the organizer: the same
// rule posts apply, so one user cannot attach another user's upload.
func validCover(ctx context.Context, d *shared.Deps, id *int64, ownerID int64) bool {
	if id == nil || *id <= 0 {
		return true
	}
	if !tableExists(d, "media") {
		return true
	}
	var mediaOwner int64
	err := d.DB.QueryRowContext(ctx, `SELECT owner_id FROM media WHERE id = ?`, *id).Scan(&mediaOwner)
	return err == nil && mediaOwner == ownerID
}

func parseEventID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, codeValidation, "Invalid event id.")
		return 0, false
	}
	return id, true
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	viewer := viewerID(r, h.d)
	p := shared.ParsePagination(r)

	var cursorStarts string
	var cursorID int64
	if p.Cursor != "" {
		var err error
		cursorStarts, cursorID, err = parseCursor(p.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "Invalid cursor."})
			return
		}
	}

	query := r.URL.Query()
	q := strings.TrimSpace(query.Get("q"))
	typ := strings.TrimSpace(query.Get("type"))
	city := strings.TrimSpace(query.Get("city"))
	country := strings.TrimSpace(query.Get("country"))
	details := map[string]string{}
	if typ != "" && !allowedTypes[typ] {
		addIssue(details, "type", "Choose a valid event type.")
	}

	var from, to string
	if raw := strings.TrimSpace(query.Get("from")); raw != "" {
		value, err := parseFilterTime(raw, false)
		if err != nil {
			addIssue(details, "from", "Use an RFC3339 timestamp or YYYY-MM-DD.")
		} else {
			from = value
		}
	}
	if raw := strings.TrimSpace(query.Get("to")); raw != "" {
		value, err := parseFilterTime(raw, true)
		if err != nil {
			addIssue(details, "to", "Use an RFC3339 timestamp or YYYY-MM-DD.")
		} else {
			to = value
		}
	}
	var online *bool
	if raw := strings.TrimSpace(query.Get("online")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			addIssue(details, "online", "Use true or false.")
		} else {
			online = &value
		}
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	where := []string{`(e.status <> 'cancelled' OR e.organizer_id = ?)`}
	// The first eventSelect placeholder is the viewer's RSVP status.
	args := []any{viewer, viewer}
	if q != "" {
		like := "%" + escapeLike(q) + "%"
		where = append(where, `(e.title LIKE ? ESCAPE '\' OR e.description LIKE ? ESCAPE '\')`)
		args = append(args, like, like)
	}
	if typ != "" {
		where = append(where, `e.event_type = ?`)
		args = append(args, typ)
	}
	if from != "" {
		where = append(where, `e.starts_at >= ?`)
		args = append(args, from)
	}
	if to != "" {
		where = append(where, `e.starts_at <= ?`)
		args = append(args, to)
	}
	if city != "" {
		where = append(where, `e.city LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(city)+"%")
	}
	if country != "" {
		where = append(where, `e.country LIKE ? ESCAPE '\'`)
		args = append(args, "%"+escapeLike(country)+"%")
	}
	if online != nil {
		where = append(where, `e.is_online = ?`)
		args = append(args, boolInt(*online))
	}
	if p.Cursor != "" {
		where = append(where, `(e.starts_at > ? OR (e.starts_at = ? AND e.id > ?))`)
		args = append(args, cursorStarts, cursorStarts, cursorID)
	}

	statement := eventSelect + ` WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY e.starts_at ASC, e.id ASC LIMIT ?`
	rows, err := h.d.DB.QueryContext(r.Context(), statement, append(args, p.Limit+1)...)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list events.")
		return
	}
	var found []eventRow
	for rows.Next() {
		row, err := scanEvent(rows)
		if err != nil {
			rows.Close()
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list events.")
			return
		}
		found = append(found, row)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not list events.")
		return
	}
	rows.Close()

	result := page[Event]{Items: []Event{}, NextCursor: ""}
	if len(found) > p.Limit {
		result.NextCursor = encodeEventCursor(found[p.Limit-1].StartsAt, found[p.Limit-1].ID)
		found = found[:p.Limit]
	}
	result.Items = buildEvents(r.Context(), h.d, found)
	shared.WriteJSON(w, http.StatusOK, result)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseEventID(w, r)
	if !ok {
		return
	}
	viewer := viewerID(r, h.d)
	row, err := eventByID(r.Context(), h.d, id, viewer)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	if !visibleEvent(row, viewer) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, buildEvents(r.Context(), h.d, []eventRow{row})[0])
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var in eventInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	v := in.normalize()
	details := map[string]string{}
	addIssue(details, "title", titleError(v.Title))
	addIssue(details, "description", descriptionError(v.Description))
	addIssue(details, "event_type", typeError(v.EventType))
	addIssue(details, "starts_at", timePairError(v.StartsAt, v.EndsAt))
	addIssue(details, "city", placeError(v.City, v.Country, v.VenueText))
	addIssue(details, "capacity", capacityError(v.Capacity))
	if !validCover(r.Context(), h.d, v.CoverMediaID, userID) {
		addIssue(details, "cover_media_id", "Cover image not found.")
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}
	startsAt, _ := parseEventTime(v.StartsAt)
	endsAt, _ := parseEventTime(v.EndsAt)

	res, err := h.d.DB.ExecContext(r.Context(), `
		INSERT INTO events(organizer_id, title, description, event_type,
		                   starts_at, ends_at, timezone, venue_text, city, country,
		                   is_online, capacity, cover_media_id, status, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,'scheduled',datetime('now'),datetime('now'))`,
		userID, v.Title, v.Description, v.EventType, startsAt, endsAt, v.Timezone,
		nullableString(v.VenueText), nullableString(v.City), nullableString(v.Country),
		boolInt(v.IsOnline), v.Capacity, v.CoverMediaID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not create the event.")
		return
	}
	id, _ := res.LastInsertId()
	row, err := eventByID(r.Context(), h.d, id, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	shared.WriteJSON(w, http.StatusCreated, buildEvents(r.Context(), h.d, []eventRow{row})[0])
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type eventPatch struct {
	Title        *string        `json:"title"`
	Description  *string        `json:"description"`
	EventType    *string        `json:"event_type"`
	EventTypeAlt *string        `json:"eventType"`
	StartsAt     *string        `json:"starts_at"`
	StartsAtAlt  *string        `json:"startsAt"`
	EndsAt       *string        `json:"ends_at"`
	EndsAtAlt    *string        `json:"endsAt"`
	Timezone     *string        `json:"timezone"`
	VenueText    optionalString `json:"venue_text"`
	VenueTextAlt optionalString `json:"venueText"`
	City         optionalString `json:"city"`
	Country      optionalString `json:"country"`
	IsOnline     *bool          `json:"is_online"`
	IsOnlineAlt  *bool          `json:"isOnline"`
	Capacity     *int           `json:"capacity"`
	CoverMediaID optionalInt64  `json:"cover_media_id"`
	CoverAlt     optionalInt64  `json:"coverMediaId"`
	Status       *string        `json:"status"`
}

func pickString(primary, alt *string) *string {
	if primary != nil {
		return primary
	}
	return alt
}

func pickOptional(primary, alt optionalString) optionalString {
	if primary.Set {
		return primary
	}
	return alt
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, ok := parseEventID(w, r)
	if !ok {
		return
	}
	current, err := loadRawEvent(r.Context(), h.d, id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	if current.OrganizerID != userID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the organiser can edit this event.")
		return
	}

	var patch eventPatch
	if !shared.DecodeJSON(w, r, &patch) {
		return
	}
	details := map[string]string{}
	sets := []string{}
	args := []any{}

	title := pickString(patch.Title, nil)
	if title != nil {
		value := strings.TrimSpace(*title)
		addIssue(details, "title", titleError(value))
		sets = append(sets, "title = ?")
		args = append(args, value)
	}
	if patch.Description != nil {
		addIssue(details, "description", descriptionError(*patch.Description))
		sets = append(sets, "description = ?")
		args = append(args, *patch.Description)
	}
	typ := pickString(patch.EventType, patch.EventTypeAlt)
	if typ != nil {
		value := strings.TrimSpace(*typ)
		addIssue(details, "event_type", typeError(value))
		sets = append(sets, "event_type = ?")
		args = append(args, value)
	}

	starts := pickString(patch.StartsAt, patch.StartsAtAlt)
	ends := pickString(patch.EndsAt, patch.EndsAtAlt)
	candidateStarts, candidateEnds := current.StartsAt, current.EndsAt
	if starts != nil {
		candidateStarts = *starts
	}
	if ends != nil {
		candidateEnds = *ends
	}
	if starts != nil || ends != nil {
		addIssue(details, "starts_at", timePairError(candidateStarts, candidateEnds))
	}
	for _, item := range []struct {
		ptr   *string
		col   string
		parse bool
	}{{starts, "starts_at", true}, {ends, "ends_at", true}} {
		if item.ptr == nil {
			continue
		}
		value := *item.ptr
		if item.parse {
			parsed, err := parseEventTime(value)
			if err == nil {
				value = parsed
			}
		}
		sets = append(sets, item.col+" = ?")
		args = append(args, value)
	}
	if patch.Timezone != nil {
		sets = append(sets, "timezone = ?")
		args = append(args, normalizeTimezone(*patch.Timezone))
	}

	venue := pickOptional(patch.VenueText, patch.VenueTextAlt)
	city := patch.City
	country := patch.Country
	for _, item := range []struct {
		opt optionalString
		col string
	}{{venue, "venue_text"}, {city, "city"}, {country, "country"}} {
		if !item.opt.Set {
			continue
		}
		value := ""
		if item.opt.Value != nil {
			value = strings.TrimSpace(*item.opt.Value)
		}
		addIssue(details, item.col, placeFieldError(item.col, value))
		sets = append(sets, item.col+" = ?")
		args = append(args, nullableOptionalString(item.opt))
	}

	online := pickBool(patch.IsOnline, patch.IsOnlineAlt)
	if online != nil {
		sets = append(sets, "is_online = ?")
		args = append(args, boolInt(*online))
	}
	if patch.Capacity != nil {
		addIssue(details, "capacity", capacityError(*patch.Capacity))
		sets = append(sets, "capacity = ?")
		args = append(args, *patch.Capacity)
	}
	cover := patch.CoverMediaID
	if !cover.Set {
		cover = patch.CoverAlt
	}
	if cover.Set {
		if cover.Value != nil && *cover.Value <= 0 {
			cover.Value = nil
		}
		if !validCover(r.Context(), h.d, cover.Value, userID) {
			addIssue(details, "cover_media_id", "Cover image not found.")
		}
		sets = append(sets, "cover_media_id = ?")
		args = append(args, cover.Value)
	}
	if patch.Status != nil {
		if !allowedStatuses[*patch.Status] {
			addIssue(details, "status", "Status must be scheduled, cancelled or completed.")
		}
		sets = append(sets, "status = ?")
		args = append(args, *patch.Status)
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	if len(sets) > 0 {
		sets = append(sets, "updated_at = datetime('now')")
		args = append(args, id)
		if _, err := h.d.DB.ExecContext(r.Context(),
			`UPDATE events SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not update the event.")
			return
		}
	}
	row, err := eventByID(r.Context(), h.d, id, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, buildEvents(r.Context(), h.d, []eventRow{row})[0])
}

// placeFieldError validates one place field of a partial patch.
func placeFieldError(col, value string) string {
	switch col {
	case "city", "country":
		if utf8.RuneCountInString(value) > maxPlaceRunes {
			return "City and country must be at most 80 characters."
		}
	case "venue_text":
		if utf8.RuneCountInString(value) > maxVenueRunes {
			return "Venue details must be at most 2000 characters."
		}
	}
	return ""
}

func nullableOptionalString(opt optionalString) any {
	if opt.Value == nil {
		return nil
	}
	return *opt.Value
}

func pickBool(primary, alt *bool) *bool {
	if primary != nil {
		return primary
	}
	return alt
}

// cancel implements DELETE as a status change so RSVPs, updates and comments
// remain available as event history.
func (h *handler) cancel(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	id, ok := parseEventID(w, r)
	if !ok {
		return
	}
	current, err := loadRawEvent(r.Context(), h.d, id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Event not found.")
		return
	}
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	if current.OrganizerID != userID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the organiser can cancel this event.")
		return
	}
	if current.Status != "cancelled" {
		if _, err := h.d.DB.ExecContext(r.Context(),
			`UPDATE events SET status = 'cancelled', updated_at = datetime('now') WHERE id = ?`, id); err != nil {
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not cancel the event.")
			return
		}
	}
	row, err := eventByID(r.Context(), h.d, id, userID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Could not load the event.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, buildEvents(r.Context(), h.d, []eventRow{row})[0])
}
