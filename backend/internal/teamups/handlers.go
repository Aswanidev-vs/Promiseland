package teamups

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

type teamUpInput struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	EventID      *int64   `json:"eventId"`
	EventIDSnake *int64   `json:"event_id"`
	Categories   []string `json:"categories"`
}

type teamUpPatch struct {
	Title        *string   `json:"title"`
	Description  *string   `json:"description"`
	EventID      *int64    `json:"eventId"`
	EventIDSnake *int64    `json:"event_id"`
	Categories   *[]string `json:"categories"`
	Status       *string   `json:"status"`
}

type applyInput struct {
	Message string `json:"message"`
}

type reviewInput struct {
	Status string `json:"status"`
}

func (i teamUpInput) eventID() *int64 {
	if i.EventID != nil {
		return i.EventID
	}
	return i.EventIDSnake
}

func (i teamUpPatch) eventID() *int64 {
	if i.EventID != nil {
		return i.EventID
	}
	return i.EventIDSnake
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation,
		"Please correct the highlighted fields.", details)
}

func parseID(w http.ResponseWriter, r *http.Request, key string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, key), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Team up not found.")
		return 0, false
	}
	return id, true
}

// checkEvent validates an optional event reference against the events table
// created by the parallel events agent. A missing table is reported as a
// validation error rather than panicking.
func (s *service) checkEvent(ctx context.Context, eventID *int64) (any, map[string]string) {
	if eventID == nil || *eventID <= 0 {
		return nil, nil
	}
	if !s.schema.HasTable("events") {
		return nil, map[string]string{"eventId": "Events are not available yet."}
	}
	var exists int
	if err := s.d.DB.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM events WHERE id = ?)`, *eventID).Scan(&exists); err != nil || exists == 0 {
		return nil, map[string]string{"eventId": "Event not found."}
	}
	return *eventID, nil
}

func (s *service) list(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.d == nil || s.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Team ups are temporarily unavailable.")
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

	q := r.URL.Query()
	status := normalize(q.Get("status"))
	if status != "" && !validStatuses[status] {
		writeValidation(w, map[string]string{"status": "Choose open, filled or closed."})
		return
	}
	category := normalize(q.Get("category"))
	if category != "" && !allowedCategories[category] {
		writeValidation(w, map[string]string{"category": "Choose a supported category."})
		return
	}
	term := strings.TrimSpace(q.Get("q"))
	if len([]rune(term)) > 100 {
		term = string([]rune(term)[:100])
	}

	viewer := viewerID(r, s.d)
	args := []any{cursor}
	query := teamUpSelect + `
		JOIN users u ON u.id = t.owner_id
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE t.id < ?`
	available, availableArgs := s.ownerAvailable(viewer)
	if available != "" {
		query += " AND " + available
		args = append(args, availableArgs...)
	}
	if status != "" {
		query += " AND t.status = ?"
		args = append(args, status)
	}
	if category != "" {
		query += ` AND instr(t.categories, ?) > 0`
		args = append(args, `"`+category+`"`)
	}
	if term != "" {
		query += ` AND (LOWER(t.title) LIKE ? ESCAPE '\' OR LOWER(t.description) LIKE ? ESCAPE '\')`
		pattern := "%" + shape.EscapeLike(strings.ToLower(term)) + "%"
		args = append(args, pattern, pattern)
	}
	query += ` ORDER BY t.id DESC LIMIT ?`
	args = append(args, p.Limit+1)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.d.DB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeInternal(w, "list team ups", err)
		return
	}
	found, err := scanTeamUpRows(rows)
	rows.Close()
	if err != nil {
		s.writeInternal(w, "scan team ups", err)
		return
	}

	result := page[TeamUp]{Items: []TeamUp{}, NextCursor: ""}
	if len(found) > p.Limit {
		found = found[:p.Limit]
		result.NextCursor = shared.EncodeIDCursor(found[len(found)-1].ID)
	}
	items, err := s.buildTeamUps(ctx, found, viewer)
	if err != nil {
		s.writeInternal(w, "build team ups", err)
		return
	}
	result.Items = items
	shared.WriteJSON(w, http.StatusOK, result)
}

func (s *service) get(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.d == nil || s.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Team ups are temporarily unavailable.")
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	viewer := viewerID(r, s.d)
	args := []any{id}
	query := teamUpSelect + `
		JOIN users u ON u.id = t.owner_id
		LEFT JOIN user_profiles up ON up.user_id = u.id
		WHERE t.id = ?`
	available, availableArgs := s.ownerAvailable(viewer)
	if available != "" {
		query += " AND " + available
		args = append(args, availableArgs...)
	}
	rows, err := s.d.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeInternal(w, "get team up", err)
		return
	}
	found, err := scanTeamUpRows(rows)
	rows.Close()
	if err != nil {
		s.writeInternal(w, "scan team up", err)
		return
	}
	if len(found) == 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Team up not found.")
		return
	}
	items, err := s.buildTeamUps(r.Context(), found, viewer)
	if err != nil || len(items) == 0 {
		s.writeInternal(w, "build team up", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, items[0])
}

func (s *service) create(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	var in teamUpInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)
	details := map[string]string{}
	if !validText(in.Title, 3, 120) {
		details["title"] = "Title must be between 3 and 120 characters."
	}
	if utf8.RuneCountInString(in.Description) > 3000 {
		details["description"] = "Description must be at most 3000 characters."
	}
	categories, categoryErrors := cleanCategories(in.Categories)
	for key, value := range categoryErrors {
		details[key] = value
	}
	eventID, eventErrors := s.checkEvent(r.Context(), in.eventID())
	for key, value := range eventErrors {
		details[key] = value
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	res, err := s.d.DB.ExecContext(r.Context(), `
		INSERT INTO team_ups(owner_id, title, description, event_id, categories, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'open', datetime('now'), datetime('now'))`,
		userID, in.Title, in.Description, eventID, encodeCategories(categories))
	if err != nil {
		s.writeInternal(w, "create team up", err)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		s.writeInternal(w, "create team up id", err)
		return
	}
	item, err := s.loadOne(r.Context(), id, userID)
	if err != nil {
		s.writeInternal(w, "load created team up", err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, item)
}

func (s *service) loadOne(ctx context.Context, id, viewerID int64) (TeamUp, error) {
	rows, err := s.d.DB.QueryContext(ctx, teamUpSelect+` WHERE t.id = ?`, id)
	if err != nil {
		return TeamUp{}, err
	}
	found, err := scanTeamUpRows(rows)
	rows.Close()
	if err != nil {
		return TeamUp{}, err
	}
	if len(found) == 0 {
		return TeamUp{}, sql.ErrNoRows
	}
	items, err := s.buildTeamUps(ctx, found, viewerID)
	if err != nil || len(items) == 0 {
		return TeamUp{}, err
	}
	return items[0], nil
}

// requireOwner returns false after writing 404 or 403.
func (s *service) requireOwner(w http.ResponseWriter, ctx context.Context, id, userID int64) bool {
	ownerID, err := s.teamUpExists(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Team up not found.")
		return false
	}
	if err != nil {
		s.writeInternal(w, "load team up owner", err)
		return false
	}
	if ownerID != userID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the team up owner can do that.")
		return false
	}
	return true
}

func (s *service) patch(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if !s.requireOwner(w, r.Context(), id, userID) {
		return
	}
	var in teamUpPatch
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	details := map[string]string{}
	if in.Title != nil {
		value := strings.TrimSpace(*in.Title)
		if !validText(value, 3, 120) {
			details["title"] = "Title must be between 3 and 120 characters."
		} else {
			in.Title = &value
		}
	}
	if in.Description != nil && utf8.RuneCountInString(*in.Description) > 3000 {
		details["description"] = "Description must be at most 3000 characters."
	}
	if in.Categories != nil {
		categories, categoryErrors := cleanCategories(*in.Categories)
		for key, value := range categoryErrors {
			details[key] = value
		}
		in.Categories = &categories
	}
	if in.Status != nil {
		status := normalize(*in.Status)
		if !validStatuses[status] {
			details["status"] = "Choose open, filled or closed."
		} else {
			in.Status = &status
		}
	}
	eventID, eventErrors := s.checkEvent(r.Context(), in.eventID())
	for key, value := range eventErrors {
		details[key] = value
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}

	sets := []string{"updated_at = datetime('now')"}
	args := []any{}
	add := func(column string, value any) {
		sets = append(sets, column+" = ?")
		args = append(args, value)
	}
	if in.Title != nil {
		add("title", *in.Title)
	}
	if in.Description != nil {
		add("description", strings.TrimSpace(*in.Description))
	}
	if in.eventID() != nil {
		if *in.eventID() <= 0 {
			add("event_id", nil)
		} else {
			add("event_id", eventID)
		}
	}
	if in.Categories != nil {
		add("categories", encodeCategories(*in.Categories))
	}
	if in.Status != nil {
		add("status", *in.Status)
	}
	args = append(args, id)
	if _, err := s.d.DB.ExecContext(r.Context(),
		`UPDATE team_ups SET `+strings.Join(sets, ", ")+` WHERE id = ?`, args...); err != nil {
		s.writeInternal(w, "update team up", err)
		return
	}
	item, err := s.loadOne(r.Context(), id, userID)
	if err != nil {
		s.writeInternal(w, "load updated team up", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, item)
}

func (s *service) remove(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if !s.requireOwner(w, r.Context(), id, userID) {
		return
	}
	if _, err := s.d.DB.ExecContext(r.Context(), `DELETE FROM team_ups WHERE id = ?`, id); err != nil {
		s.writeInternal(w, "delete team up", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (s *service) apply(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ownerID, err := s.teamUpExists(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Team up not found.")
		return
	}
	if err != nil {
		s.writeInternal(w, "apply: load team up", err)
		return
	}
	if ownerID == userID {
		shared.WriteError(w, http.StatusBadRequest, codeSelfApply, "You cannot apply to your own team up.")
		return
	}
	var in applyInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	in.Message = strings.TrimSpace(in.Message)
	if utf8.RuneCountInString(in.Message) > 3000 {
		writeValidation(w, map[string]string{"message": "Message must be at most 3000 characters."})
		return
	}
	res, err := s.d.DB.ExecContext(r.Context(), `
		INSERT OR IGNORE INTO team_up_applications(team_up_id, applicant_id, message, status, created_at)
		VALUES (?, ?, ?, 'pending', datetime('now'))`, id, userID, in.Message)
	if err != nil {
		s.writeInternal(w, "apply insert", err)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		shared.WriteError(w, http.StatusConflict, codeApplied, "You have already applied to this team up.")
		return
	}
	application, err := s.loadApplication(r.Context(), id, userID)
	if err != nil {
		s.writeInternal(w, "apply load", err)
		return
	}
	s.notify(ownerID, userID, "team_up_application", "team_up", id, `{"application_id":`+
		strconv.FormatInt(application.ID, 10)+`}`)
	shared.WriteJSON(w, http.StatusCreated, application)
}

func (s *service) listApplications(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if !s.requireOwner(w, r.Context(), id, userID) {
		return
	}
	rows, err := s.d.DB.QueryContext(r.Context(), `
		SELECT id, team_up_id, applicant_id, message, status, created_at
		FROM team_up_applications WHERE team_up_id = ? ORDER BY id`, id)
	if err != nil {
		s.writeInternal(w, "list applications", err)
		return
	}
	items := []TeamUpApplication{}
	applicantIDs := []int64{}
	type applicationRow struct {
		id, teamUpID, applicantID int64
		message, status, created  string
	}
	found := []applicationRow{}
	for rows.Next() {
		var row applicationRow
		if err := rows.Scan(&row.id, &row.teamUpID, &row.applicantID, &row.message, &row.status, &row.created); err != nil {
			rows.Close()
			s.writeInternal(w, "scan applications", err)
			return
		}
		found = append(found, row)
		applicantIDs = append(applicantIDs, row.applicantID)
	}
	rows.Close()
	applicants := s.loadUsers(r.Context(), applicantIDs)
	for _, row := range found {
		applicant := applicants[row.applicantID]
		if applicant.ID == 0 {
			applicant = UserSummary{ID: row.applicantID, DisplayName: "unknown"}
		}
		items = append(items, TeamUpApplication{
			ID: row.id, TeamUpID: row.teamUpID, Applicant: applicant,
			Message: row.message, Status: row.status, CreatedAt: row.created,
		})
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

func (s *service) loadApplication(ctx context.Context, teamUpID, applicantID int64) (TeamUpApplication, error) {
	var item TeamUpApplication
	var applicantIDValue int64
	err := s.d.DB.QueryRowContext(ctx, `
		SELECT id, team_up_id, applicant_id, message, status, created_at
		FROM team_up_applications WHERE team_up_id = ? AND applicant_id = ?`,
		teamUpID, applicantID).Scan(&item.ID, &item.TeamUpID, &applicantIDValue,
		&item.Message, &item.Status, &item.CreatedAt)
	if err != nil {
		return TeamUpApplication{}, err
	}
	applicants := s.loadUsers(ctx, []int64{applicantIDValue})
	item.Applicant = applicants[applicantIDValue]
	if item.Applicant.ID == 0 {
		item.Applicant = UserSummary{ID: applicantIDValue, DisplayName: "unknown"}
	}
	return item, nil
}

func (s *service) reviewApplication(w http.ResponseWriter, r *http.Request) {
	userID, ok := shared.UserID(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	teamUpID, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	applicationID, ok := parseID(w, r, "applicationId")
	if !ok {
		return
	}
	var in reviewInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	status := normalize(in.Status)
	if status != "accepted" && status != "rejected" {
		writeValidation(w, map[string]string{"status": "Choose accepted or rejected."})
		return
	}

	tx, err := s.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		s.writeInternal(w, "review begin", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var ownerID int64
	if err := tx.QueryRowContext(r.Context(),
		`SELECT owner_id FROM team_ups WHERE id = ?`, teamUpID).Scan(&ownerID); errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Team up not found.")
		return
	} else if err != nil {
		s.writeInternal(w, "review load team up", err)
		return
	}
	if ownerID != userID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden, "Only the team up owner can review applications.")
		return
	}
	result, err := tx.ExecContext(r.Context(), `
		UPDATE team_up_applications SET status = ? WHERE id = ? AND team_up_id = ?`,
		status, applicationID, teamUpID)
	if err != nil {
		s.writeInternal(w, "review update", err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Application not found.")
		return
	}
	if err := tx.Commit(); err != nil {
		s.writeInternal(w, "review commit", err)
		return
	}

	var applicantID int64
	if err := s.d.DB.QueryRowContext(r.Context(),
		`SELECT applicant_id FROM team_up_applications WHERE id = ? AND team_up_id = ?`,
		applicationID, teamUpID).Scan(&applicantID); err != nil {
		s.writeInternal(w, "review applicant", err)
		return
	}
	application, err := s.loadApplication(r.Context(), teamUpID, applicantID)
	if err != nil {
		s.writeInternal(w, "review load application", err)
		return
	}
	s.notify(applicantID, userID, "team_up_application_review", "team_up", teamUpID,
		`{"status":"`+status+`","application_id":`+strconv.FormatInt(applicationID, 10)+`}`)
	shared.WriteJSON(w, http.StatusOK, application)
}
