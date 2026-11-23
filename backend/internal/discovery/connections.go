package discovery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

var validConnectionStatuses = map[string]bool{
	"pending": true, "accepted": true, "rejected": true, "ignored": true,
}

var validReviewStatuses = map[string]bool{
	"accepted": true, "rejected": true, "ignored": true,
}

// loadConnection resolves the exact R4 Connection shape for one row.
func (m *module) loadConnection(ctx context.Context, id int64) (Connection, error) {
	var connection Connection
	var requesterID, addresseeID int64
	var message, status, createdAt, updatedAt string
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT id, requester_id, addressee_id, COALESCE(message, ''), status, created_at, updated_at
		FROM connections WHERE id = ?`, id).
		Scan(&connection.ID, &requesterID, &addresseeID, &message, &status, &createdAt, &updatedAt)
	if err != nil {
		return Connection{}, err
	}
	connection.Message = message
	connection.Status = status
	connection.CreatedAt = createdAt
	connection.UpdatedAt = updatedAt
	summaries, err := m.loadUserSummaries(ctx, []int64{requesterID, addresseeID})
	if err != nil {
		return Connection{}, err
	}
	connection.Requester = summaries[requesterID]
	connection.Addressee = summaries[addresseeID]
	return connection, nil
}

// notify is intentionally best-effort: a notification failure must never
// roll back the connection change that caused it.
func (m *module) notify(ctx context.Context, userID int64, typ string, actorID int64, targetType string, targetID int64) {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
		return
	}
	_, err := m.d.DB.ExecContext(ctx, `
		INSERT INTO notifications (user_id, type, actor_id, target_type, target_id, data, created_at)
		VALUES (?, ?, ?, ?, ?, '{}', datetime('now'))`,
		userID, typ, actorID, targetType, targetID)
	if err != nil && m.d.Log != nil {
		m.d.Log.Warn("discovery: could not create connection notification",
			"error", err, "type", typ, "user_id", userID)
	}
}

func (m *module) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	input, ok := decodeConnectionInput(w, r)
	if !ok {
		return
	}
	if input.UserID <= 0 {
		writeValidation(w, "Invalid connection request.", fieldErrors{
			"userId": "must be a positive integer user id",
		})
		return
	}
	if input.UserID == requesterID {
		writeValidation(w, "Invalid connection request.", fieldErrors{
			"userId": "you cannot connect with yourself",
		})
		return
	}
	exists, err := m.userExists(r.Context(), input.UserID)
	if err != nil {
		internalError(w, m.d, "create connection: check target", err)
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	blocked, err := blockedEitherWay(r.Context(), m.d.DB, requesterID, input.UserID)
	if err != nil {
		internalError(w, m.d, "create connection: check block", err)
		return
	}
	if blocked {
		shared.WriteError(w, http.StatusForbidden, codeBlocked, "You cannot connect with this user.")
		return
	}
	var discoverable int
	var visibility string
	err = m.d.DB.QueryRowContext(r.Context(), `
		SELECT u.discoverable, COALESCE(p.visibility, 'public')
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id = ? AND u.is_active = 1`, input.UserID).Scan(&discoverable, &visibility)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "User not found.")
		return
	}
	if err != nil {
		internalError(w, m.d, "create connection: load target", err)
		return
	}
	if visibility == "private" {
		shared.WriteError(w, http.StatusForbidden, codeProfilePrivate, "This profile is private.")
		return
	}
	if discoverable == 0 {
		shared.WriteError(w, http.StatusForbidden, "not_discoverable", "This user is not accepting discovery requests.")
		return
	}

	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, m.d, "create connection: begin", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var existingID int64
	err = tx.QueryRowContext(r.Context(), `
		SELECT id FROM connections
		WHERE (requester_id = ? AND addressee_id = ?)
		   OR (requester_id = ? AND addressee_id = ?)
		LIMIT 1`, requesterID, input.UserID, input.UserID, requesterID).Scan(&existingID)
	if err == nil {
		shared.WriteError(w, http.StatusConflict, codeConnection,
			"A connection already exists between these users.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		internalError(w, m.d, "create connection: check duplicate", err)
		return
	}
	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO connections (requester_id, addressee_id, message, status)
		VALUES (?, ?, ?, 'pending')`, requesterID, input.UserID, input.Message)
	if err != nil {
		internalError(w, m.d, "create connection: insert", err)
		return
	}
	connectionID, err := result.LastInsertId()
	if err != nil {
		internalError(w, m.d, "create connection: id", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, m.d, "create connection: commit", err)
		return
	}

	connection, err := m.loadConnection(r.Context(), connectionID)
	if err != nil {
		internalError(w, m.d, "create connection: reload", err)
		return
	}
	m.notify(r.Context(), input.UserID, "connection_request", requesterID, "connection", connectionID)
	shared.WriteJSON(w, http.StatusCreated, connection)
}

func (m *module) handleListConnections(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	query := r.URL.Query()
	status := strings.TrimSpace(query.Get("status"))
	if status != "" && !validConnectionStatuses[status] {
		writeValidation(w, "Invalid connection filter.", fieldErrors{
			"status": "must be pending, accepted, rejected or ignored",
		})
		return
	}
	direction := strings.TrimSpace(query.Get("direction"))
	if direction == "" {
		direction = "all"
	}
	if direction != "all" && direction != "incoming" && direction != "outgoing" {
		writeValidation(w, "Invalid connection filter.", fieldErrors{
			"direction": "must be incoming, outgoing or all",
		})
		return
	}
	pg := shared.ParsePagination(r)
	var afterID int64
	if pg.Cursor != "" {
		var err error
		afterID, err = shared.DecodeIDCursor(pg.Cursor)
		if err != nil {
			writeValidation(w, "Invalid pagination cursor.", fieldErrors{
				"cursor": "must be a valid cursor",
			})
			return
		}
	}

	args := make([]any, 0, 5)
	var where string
	switch direction {
	case "incoming":
		where = "c.addressee_id = ?"
		args = append(args, viewerID)
	case "outgoing":
		where = "c.requester_id = ?"
		args = append(args, viewerID)
	default:
		where = "(c.requester_id = ? OR c.addressee_id = ?)"
		args = append(args, viewerID, viewerID)
	}
	if status != "" {
		where += " AND c.status = ?"
		args = append(args, status)
	}
	if afterID > 0 {
		where += " AND c.id < ?"
		args = append(args, afterID)
	}
	args = append(args, pg.Limit+1)
	rows, err := m.d.DB.QueryContext(r.Context(), fmt.Sprintf(`
		SELECT c.id, c.requester_id, c.addressee_id,
		       COALESCE(c.message, ''), c.status, c.created_at, c.updated_at
		FROM connections c
		WHERE %s
		ORDER BY c.id DESC
		LIMIT ?`, where), args...)
	if err != nil {
		internalError(w, m.d, "list connections", err)
		return
	}

	type connectionRow struct {
		id                                    int64
		requesterID, addresseeID              int64
		message, status, createdAt, updatedAt string
	}
	found := make([]connectionRow, 0, pg.Limit+1)
	for rows.Next() {
		var item connectionRow
		if err := rows.Scan(&item.id, &item.requesterID, &item.addresseeID,
			&item.message, &item.status, &item.createdAt, &item.updatedAt); err != nil {
			_ = rows.Close()
			internalError(w, m.d, "scan connection", err)
			return
		}
		found = append(found, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		internalError(w, m.d, "iterate connections", err)
		return
	}
	_ = rows.Close()

	ids := make([]int64, 0, len(found))
	for _, item := range found {
		ids = append(ids, item.id)
	}
	var summaries map[int64]UserSummary
	if len(ids) > 0 {
		userIDs := make([]int64, 0, 2*len(ids))
		for _, item := range found {
			userIDs = append(userIDs, item.requesterID, item.addresseeID)
		}
		summaries, err = m.loadUserSummaries(r.Context(), userIDs)
		if err != nil {
			internalError(w, m.d, "list connections: load users", err)
			return
		}
	}

	items := make([]Connection, 0, pg.Limit)
	nextCursor := ""
	for index, item := range found {
		if index == pg.Limit {
			nextCursor = shared.EncodeIDCursor(found[pg.Limit-1].id)
			break
		}
		items = append(items, Connection{
			ID:        item.id,
			Requester: summaries[item.requesterID],
			Addressee: summaries[item.addresseeID],
			Message:   item.message,
			Status:    item.status,
			CreatedAt: item.createdAt,
			UpdatedAt: item.updatedAt,
		})
	}
	shared.WriteJSON(w, http.StatusOK, page[Connection]{Items: items, NextCursor: nextCursor})
}

func (m *module) handleReviewConnection(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	id, ok := parsePositiveID(r, "id")
	if !ok {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid connection id.")
		return
	}
	raw, ok := decodeFields(w, r)
	if !ok {
		return
	}
	status, valid := decodeString(raw["status"])
	if !valid || !validReviewStatuses[status] {
		writeValidation(w, "Invalid connection status.", fieldErrors{
			"status": "must be accepted, rejected or ignored",
		})
		return
	}

	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, m.d, "review connection: begin", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var requesterID, addresseeID int64
	var currentStatus string
	err = tx.QueryRowContext(r.Context(),
		`SELECT requester_id, addressee_id, status FROM connections WHERE id = ?`, id).
		Scan(&requesterID, &addresseeID, &currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Connection not found.")
		return
	}
	if err != nil {
		internalError(w, m.d, "review connection: load", err)
		return
	}
	if addresseeID != viewerID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden,
			"Only the recipient can review this connection request.")
		return
	}
	if currentStatus != "pending" {
		shared.WriteError(w, http.StatusConflict, codeNotPending,
			"Only a pending connection request can be reviewed.")
		return
	}
	if _, err := tx.ExecContext(r.Context(),
		`UPDATE connections SET status = ?, updated_at = datetime('now') WHERE id = ?`, status, id); err != nil {
		internalError(w, m.d, "review connection: update", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, m.d, "review connection: commit", err)
		return
	}

	if status == "accepted" {
		m.notify(r.Context(), requesterID, "connection_accepted", viewerID, "connection", id)
	}
	connection, err := m.loadConnection(r.Context(), id)
	if err != nil {
		internalError(w, m.d, "review connection: reload", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, connection)
}

func (m *module) handleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := requireViewer(w, r, m)
	if !ok {
		return
	}
	id, ok := parsePositiveID(r, "id")
	if !ok {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid connection id.")
		return
	}
	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalError(w, m.d, "delete connection: begin", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var requesterID, addresseeID int64
	err = tx.QueryRowContext(r.Context(),
		`SELECT requester_id, addressee_id FROM connections WHERE id = ?`, id).
		Scan(&requesterID, &addresseeID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Connection not found.")
		return
	}
	if err != nil {
		internalError(w, m.d, "delete connection: load", err)
		return
	}
	if requesterID != viewerID && addresseeID != viewerID {
		shared.WriteError(w, http.StatusForbidden, codeForbidden,
			"You are not a party to this connection.")
		return
	}
	if _, err := tx.ExecContext(r.Context(), `DELETE FROM connections WHERE id = ?`, id); err != nil {
		internalError(w, m.d, "delete connection: delete", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, m.d, "delete connection: commit", err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, nil)
}
