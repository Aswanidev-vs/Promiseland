// Package moderation implements R1 reporting, platform moderation and the
// moderation audit log. All write endpoints enforce session authentication and
// platform-moderator authorization in the handler itself.
package moderation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/feed/shape"
	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	errValidation    = "validation_error"
	errNotFound      = "not_found"
	errForbidden     = "forbidden"
	errAlreadyReport = "already_reported"
	maxReasonLen     = 500
	maxDetailsLen    = 2000
)

var validReasons = map[string]bool{
	"spam": true, "harassment": true, "impersonation": true,
	"hate_abuse": true, "inappropriate": true, "copyright": true,
	"spoiler_abuse": true, "unsafe_event": true, "other": true,
}

var validTargetTypes = map[string]bool{
	"post": true, "posts": true, "comment": true, "comments": true,
	"user": true, "users": true, "community": true, "communities": true,
	"event": true, "events": true, "message": true, "messages": true,
}

type module struct {
	d *shared.Deps
	s *shape.Schema
}

// Register mounts moderation routes on a public router. Handlers perform
// their own authentication and authorization checks.
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
	m := &module{d: d, s: shape.LoadSchema(ctx, db)}
	m.bootstrapModerator(ctx)
	return m
}

func (m *module) mount(r chi.Router) {
	if r == nil {
		return
	}
	r.Post("/reports", m.handleCreateReport)
	r.Get("/moderation/reports", m.handleListReports)
	r.Post("/moderation/reports/{id}/resolve", m.handleResolveReport)
	r.Get("/moderation/actions", m.handleListActions)
	r.Post("/users/{id}/suspend", m.handleSuspendUser)
}

// bootstrapModerator applies the optional dev-only PLATFORM_MODERATOR_EMAIL
// grant. Production ignores the variable so it cannot create a backdoor.
func (m *module) bootstrapModerator(ctx context.Context) {
	if m == nil || m.d == nil || m.d.DB == nil || m.d.Config.IsProduction() {
		return
	}
	email := strings.TrimSpace(os.Getenv("PLATFORM_MODERATOR_EMAIL"))
	if email == "" || !m.s.HasTable("users") || !m.s.HasTable("platform_moderators") {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var userID int64
	if err := m.d.DB.QueryRowContext(ctx, `SELECT id FROM users WHERE email = ? LIMIT 1`, email).Scan(&userID); err != nil {
		return
	}
	_, _ = m.d.DB.ExecContext(ctx, `INSERT OR IGNORE INTO platform_moderators (user_id) VALUES (?)`, userID)
	if m.s.HasCol("users", "is_platform_moderator") {
		_, _ = m.d.DB.ExecContext(ctx, `UPDATE users SET is_platform_moderator = 1 WHERE id = ?`, userID)
	}
	if m.s.HasCol("users", "is_moderator") {
		_, _ = m.d.DB.ExecContext(ctx, `UPDATE users SET is_moderator = 1 WHERE id = ?`, userID)
	}
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

func (m *module) requireModerator(w http.ResponseWriter, r *http.Request) (int64, bool) {
	viewerID, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return 0, false
	}
	granted, err := m.isModerator(r.Context(), viewerID)
	if err != nil {
		m.logError("moderator check failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Authorization could not be verified.")
		return 0, false
	}
	if !granted {
		shared.WriteError(w, http.StatusForbidden, errForbidden, "Platform moderator access required.")
		return 0, false
	}
	return viewerID, true
}

func (m *module) isModerator(ctx context.Context, userID int64) (bool, error) {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
		return false, nil
	}
	if m.s.HasTable("platform_moderators") {
		var one int
		err := m.d.DB.QueryRowContext(ctx, `SELECT 1 FROM platform_moderators WHERE user_id = ? LIMIT 1`, userID).Scan(&one)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	for _, column := range []string{"is_platform_moderator", "is_moderator"} {
		if !m.s.HasCol("users", column) {
			continue
		}
		var enabled int
		err := m.d.DB.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT COALESCE(%s, 0) FROM users WHERE id = ?`, column), userID).Scan(&enabled)
		if err == nil && enabled != 0 {
			return true, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	if m.s.HasCol("users", "is_admin") {
		var enabled int
		err := m.d.DB.QueryRowContext(ctx, `SELECT COALESCE(is_admin, 0) FROM users WHERE id = ?`, userID).Scan(&enabled)
		if err == nil && enabled != 0 {
			return true, nil
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	if m.s.HasCol("users", "role") {
		var role string
		err := m.d.DB.QueryRowContext(ctx, `SELECT COALESCE(role, '') FROM users WHERE id = ?`, userID).Scan(&role)
		if err == nil {
			switch strings.ToLower(role) {
			case "admin", "platform_moderator", "platform-moderator", "moderator":
				return true, nil
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
	}
	return false, nil
}

// Report is the exact R1 Report wire shape.
type Report struct {
	ID         int64   `json:"id"`
	ReporterID int64   `json:"reporter_id"`
	TargetType string  `json:"target_type"`
	TargetID   int64   `json:"target_id"`
	Reason     string  `json:"reason"`
	Details    *string `json:"details"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
}

type reportRequest struct {
	TargetType string `json:"targetType"`
	TargetID   int64  `json:"targetId"`
	Reason     string `json:"reason"`
	Details    string `json:"details"`
}

func (m *module) handleCreateReport(w http.ResponseWriter, r *http.Request) {
	reporterID, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return
	}
	if !m.s.HasTable("reports") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Reporting is temporarily unavailable.")
		return
	}
	var body reportRequest
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	body.TargetType = strings.ToLower(strings.TrimSpace(body.TargetType))
	body.Reason = strings.ToLower(strings.TrimSpace(body.Reason))
	details := map[string]string{}
	if !validTargetTypes[body.TargetType] {
		details["targetType"] = "unsupported report target"
	}
	if body.TargetID <= 0 {
		details["targetId"] = "must be a positive integer"
	}
	if !validReasons[body.Reason] {
		details["reason"] = "unsupported report reason"
	}
	if len(body.Details) > maxDetailsLen {
		details["details"] = "must be at most 2000 characters"
	}
	if len(details) > 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation, "Invalid report.", details)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var existing int
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM reports
		WHERE reporter_id = ? AND target_type = ? AND target_id = ? AND status = 'open'
		LIMIT 1`, reporterID, body.TargetType, body.TargetID).Scan(&existing)
	if err == nil {
		shared.WriteError(w, http.StatusConflict, errAlreadyReport, "You have already reported this target.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		m.logError("duplicate report check failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be submitted.")
		return
	}

	detailsValue := any(nil)
	if strings.TrimSpace(body.Details) != "" {
		detailsValue = strings.TrimSpace(body.Details)
	}
	res, err := m.d.DB.ExecContext(ctx, `
		INSERT INTO reports (reporter_id, target_type, target_id, reason, details, status)
		VALUES (?, ?, ?, ?, ?, 'open')`,
		reporterID, body.TargetType, body.TargetID, body.Reason, detailsValue)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			shared.WriteError(w, http.StatusConflict, errAlreadyReport, "You have already reported this target.")
			return
		}
		m.logError("create report failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be submitted.")
		return
	}
	id, _ := res.LastInsertId()
	var createdAt string
	var storedDetails sql.NullString
	var status string
	err = m.d.DB.QueryRowContext(ctx,
		`SELECT created_at, details, status FROM reports WHERE id = ?`, id).
		Scan(&createdAt, &storedDetails, &status)
	if err != nil {
		m.logError("reload report failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be submitted.")
		return
	}
	report := Report{
		ID: id, ReporterID: reporterID, TargetType: body.TargetType, TargetID: body.TargetID,
		Reason: body.Reason, Status: status, CreatedAt: createdAt,
	}
	if storedDetails.Valid {
		v := storedDetails.String
		report.Details = &v
	}
	shared.WriteJSON(w, http.StatusCreated, report)
}

func (m *module) handleListReports(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireModerator(w, r); !ok {
		return
	}
	if !m.s.HasTable("reports") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Reports are temporarily unavailable.")
		return
	}
	pg := shared.ParsePagination(r)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	if statusFilter != "" && statusFilter != "open" && statusFilter != "resolved" && statusFilter != "dismissed" {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid report status.", map[string]string{"status": "must be open, resolved or dismissed"})
		return
	}
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
	clauses := []string{"1 = 1"}
	args := []any{}
	if statusFilter != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, statusFilter)
	}
	if afterID > 0 {
		clauses = append(clauses, "id < ?")
		args = append(args, afterID)
	}
	q := fmt.Sprintf(`SELECT id, reporter_id, target_type, target_id, reason, details, status, created_at
		FROM reports WHERE %s ORDER BY id DESC LIMIT ?`, strings.Join(clauses, " AND "))
	args = append(args, pg.Limit+1)
	rows, err := m.d.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		m.logError("list reports failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Reports could not be loaded.")
		return
	}
	defer rows.Close()
	items := []Report{}
	for rows.Next() {
		var rep Report
		var details sql.NullString
		if err := rows.Scan(&rep.ID, &rep.ReporterID, &rep.TargetType, &rep.TargetID,
			&rep.Reason, &details, &rep.Status, &rep.CreatedAt); err != nil {
			m.logError("scan report failed", err)
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Reports could not be loaded.")
			return
		}
		if details.Valid {
			v := details.String
			rep.Details = &v
		}
		items = append(items, rep)
	}
	if err := rows.Err(); err != nil {
		m.logError("iterate reports failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Reports could not be loaded.")
		return
	}
	page := shared.NewPage(items)
	if len(items) > pg.Limit {
		page.Items = items[:pg.Limit]
		page.NextCursor = shared.EncodeIDCursor(items[pg.Limit-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, page)
}

type resolveRequest struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func (m *module) handleResolveReport(w http.ResponseWriter, r *http.Request) {
	moderatorID, ok := m.requireModerator(w, r)
	if !ok {
		return
	}
	reportID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || reportID <= 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid report id.", map[string]string{"id": "must be a positive integer"})
		return
	}
	var body resolveRequest
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	body.Action = strings.ToLower(strings.TrimSpace(body.Action))
	body.Reason = strings.TrimSpace(body.Reason)
	details := map[string]string{}
	// The UI uses remove_content/warn_user; the original API contract uses
	// remove/suspend. Both vocabularies are accepted and normalised before any
	// state change or audit entry is written.
	canonicalAction := body.Action
	switch body.Action {
	case "remove", "remove_content":
		canonicalAction = "remove"
	case "warn_user":
		canonicalAction = "warn"
	case "dismiss", "suspend":
	default:
		details["action"] = "must be remove, remove_content, warn_user, dismiss or suspend"
	}
	if body.Reason == "" || len(body.Reason) > maxReasonLen {
		details["reason"] = "must be between 1 and 500 characters"
	}
	if len(details) > 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation, "Invalid resolution.", details)
		return
	}
	if !m.s.HasTable("reports") || !m.s.HasTable("moderation_actions") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Moderation is temporarily unavailable.")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	tx, err := m.d.DB.BeginTx(ctx, nil)
	if err != nil {
		m.logError("begin resolve failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be resolved.")
		return
	}
	defer func() { _ = tx.Rollback() }()

	var targetType, reason string
	var targetID int64
	err = tx.QueryRowContext(ctx,
		`SELECT target_type, target_id, reason FROM reports WHERE id = ?`, reportID).
		Scan(&targetType, &targetID, &reason)
	if err != nil {
		shared.WriteError(w, http.StatusNotFound, errNotFound, "Report not found.")
		return
	}

	status := "resolved"
	auditTargetType, auditTargetID := targetType, targetID
	switch canonicalAction {
	case "remove":
		m.softDeleteTarget(ctx, tx, targetType, targetID)
	case "warn":
		// A warning is recorded against the target author when the report is
		// about content, or against the reported account directly.
		userID := targetID
		if targetType != "user" && targetType != "users" {
			userID = m.authorOfTarget(ctx, tx, targetType, targetID)
		}
		if userID > 0 {
			auditTargetType, auditTargetID = "user", userID
		}
	case "suspend":
		userID := targetID
		if targetType != "user" && targetType != "users" {
			userID = m.authorOfTarget(ctx, tx, targetType, targetID)
		}
		if userID > 0 {
			m.suspend(ctx, tx, userID, body.Reason)
		}
	case "dismiss":
		status = "dismissed"
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE reports SET status = ?, resolved_at = datetime('now'), resolved_by = ?
		WHERE id = ?`, status, moderatorID, reportID); err != nil {
		m.logError("update report failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be resolved.")
		return
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO moderation_actions
			(moderator_id, scope, target_type, target_id, action, reason, created_at)
		VALUES (?, 'platform', ?, ?, ?, ?, datetime('now'))`,
		moderatorID, auditTargetType, auditTargetID, canonicalAction, body.Reason); err != nil {
		m.logError("record moderation action failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be resolved.")
		return
	}
	if err := tx.Commit(); err != nil {
		m.logError("commit resolve failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be resolved.")
		return
	}
	updated, err := m.reportByID(r.Context(), reportID)
	if err != nil {
		m.logError("reload resolved report failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Report could not be resolved.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, updated)
}

func (m *module) reportByID(ctx context.Context, id int64) (Report, error) {
	var rep Report
	var details sql.NullString
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT id, reporter_id, target_type, target_id, reason, details, status, created_at
		FROM reports WHERE id = ?`, id).
		Scan(&rep.ID, &rep.ReporterID, &rep.TargetType, &rep.TargetID,
			&rep.Reason, &details, &rep.Status, &rep.CreatedAt)
	if err != nil {
		return rep, err
	}
	if details.Valid {
		v := details.String
		rep.Details = &v
	}
	return rep, nil
}

// softDeleteTarget sets deleted_at when the target table supports it.
func (m *module) softDeleteTarget(ctx context.Context, tx *sql.Tx, targetType string, targetID int64) {
	table := ""
	switch strings.ToLower(targetType) {
	case "post", "posts":
		table = "posts"
	case "comment", "comments":
		table = "comments"
	default:
		return
	}
	if !m.s.HasTable(table) || !m.s.HasCol(table, "deleted_at") {
		return
	}
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET deleted_at = datetime('now') WHERE id = ? AND deleted_at IS NULL`, table), targetID)
	if err != nil {
		m.logError("soft delete target failed", err)
	}
}

func (m *module) authorOfTarget(ctx context.Context, tx *sql.Tx, targetType string, targetID int64) int64 {
	table := ""
	col := ""
	switch strings.ToLower(targetType) {
	case "post", "posts":
		table, col = "posts", "author_id"
	case "comment", "comments":
		table, col = "comments", "author_id"
	}
	if table == "" || !m.s.HasTable(table) || !m.s.HasCol(table, col) {
		return 0
	}
	var id int64
	if tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT %s FROM %s WHERE id = ?`, col, table), targetID).Scan(&id) != nil {
		return 0
	}
	return id
}

func (m *module) suspend(ctx context.Context, tx *sql.Tx, userID int64, reason string) {
	if !m.s.HasTable("suspensions") {
		return
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO suspensions (user_id, reason, expires_at, is_active)
		VALUES (?, ?, NULL, 1)`, userID, reason); err != nil {
		m.logError("insert suspension failed", err)
		return
	}
	if m.s.HasCol("users", "is_active") {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET is_active = 0 WHERE id = ?`, userID); err != nil {
			m.logError("deactivate suspended user failed", err)
		}
	}
	// Drop existing sessions so the suspension takes effect immediately.
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		m.logError("revoke sessions of suspended user failed", err)
	}
}

func (m *module) handleSuspendUser(w http.ResponseWriter, r *http.Request) {
	moderatorID, ok := m.requireModerator(w, r)
	if !ok {
		return
	}
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || userID <= 0 {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid user id.", map[string]string{"id": "must be a positive integer"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if !shared.DecodeJSON(w, r, &body) {
		return
	}
	body.Reason = strings.TrimSpace(body.Reason)
	if body.Reason == "" || len(body.Reason) > maxReasonLen {
		shared.WriteErrorDetails(w, http.StatusBadRequest, errValidation,
			"Invalid suspension reason.", map[string]string{"reason": "must be between 1 and 500 characters"})
		return
	}
	if !m.s.HasTable("suspensions") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Moderation is temporarily unavailable.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	tx, err := m.d.DB.BeginTx(ctx, nil)
	if err != nil {
		m.logError("begin suspension failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Account could not be suspended.")
		return
	}
	defer func() { _ = tx.Rollback() }()
	m.suspend(ctx, tx, userID, body.Reason)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO moderation_actions
			(moderator_id, scope, target_type, target_id, action, reason, created_at)
		VALUES (?, 'platform', 'user', ?, 'suspend', ?, datetime('now'))`,
		moderatorID, userID, body.Reason); err != nil {
		m.logError("record suspension failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Account could not be suspended.")
		return
	}
	if err := tx.Commit(); err != nil {
		m.logError("commit suspension failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Account could not be suspended.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ModerationAction is one audit-log entry.
type ModerationAction struct {
	ID          int64   `json:"id"`
	ModeratorID *int64  `json:"moderator_id"`
	Scope       string  `json:"scope"`
	TargetType  string  `json:"target_type"`
	TargetID    int64   `json:"target_id"`
	Action      string  `json:"action"`
	Reason      *string `json:"reason"`
	CreatedAt   string  `json:"created_at"`
}

func (m *module) handleListActions(w http.ResponseWriter, r *http.Request) {
	if _, ok := m.requireModerator(w, r); !ok {
		return
	}
	if !m.s.HasTable("moderation_actions") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Moderation log is temporarily unavailable.")
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
	clauses := []string{"1 = 1"}
	args := []any{}
	if afterID > 0 {
		clauses = append(clauses, "id < ?")
		args = append(args, afterID)
	}
	args = append(args, pg.Limit+1)
	q := fmt.Sprintf(`SELECT id, moderator_id, scope, target_type, target_id, action, reason, created_at
		FROM moderation_actions WHERE %s ORDER BY id DESC LIMIT ?`, strings.Join(clauses, " AND "))
	rows, err := m.d.DB.QueryContext(r.Context(), q, args...)
	if err != nil {
		m.logError("list moderation actions failed", err)
		shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Moderation log could not be loaded.")
		return
	}
	defer rows.Close()
	items := []ModerationAction{}
	for rows.Next() {
		var a ModerationAction
		var moderatorID sql.NullInt64
		var reason sql.NullString
		if err := rows.Scan(&a.ID, &moderatorID, &a.Scope, &a.TargetType, &a.TargetID,
			&a.Action, &reason, &a.CreatedAt); err != nil {
			m.logError("scan moderation action failed", err)
			shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "Moderation log could not be loaded.")
			return
		}
		if moderatorID.Valid {
			v := moderatorID.Int64
			a.ModeratorID = &v
		}
		if reason.Valid {
			v := reason.String
			a.Reason = &v
		}
		items = append(items, a)
	}
	page := shared.NewPage(items)
	if len(items) > pg.Limit {
		page.Items = items[:pg.Limit]
		page.NextCursor = shared.EncodeIDCursor(items[pg.Limit-1].ID)
	}
	shared.WriteJSON(w, http.StatusOK, page)
}

func (m *module) logError(message string, err error) {
	if m != nil && m.d != nil && m.d.Log != nil {
		m.d.Log.Warn(message, "error", err)
	}
}
