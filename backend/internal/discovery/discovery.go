package discovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeValidation     = "validation_error"
	codeInvalidID      = "invalid_id"
	codeBlocked        = "blocked"
	codeProfilePrivate = "profile_private"
	codeConnection     = "connection_exists"
	codeForbidden      = "forbidden"
	codeNotPending     = "connection_not_pending"

	// candidateCap bounds the candidate scan so a large user table cannot turn
	// one discovery request into a full-table overlap calculation.
	candidateCap = 2000
	// queryChunk keeps SQL parameter counts well below SQLite's variable limit.
	queryChunk = 400
	// sharedNameCap is the maximum number of readable names per shared array.
	sharedNameCap = 5
	// maxMessageRunes bounds optional connection request messages.
	maxMessageRunes = 500
)

type module struct {
	d *shared.Deps
}

// Register mounts the discovery and connection routes on a router. Handlers
// resolve and verify the viewer themselves (shared.UserID first, session
// cookie fallback), so these routes are protected even when wired on a public
// subrouter. The orchestrator may also mount RegisterProtected on a router
// wrapped with shared.RequireAuth without changing behaviour.
func Register(r chi.Router, d *shared.Deps) {
	mount(r, d)
}

// RegisterProtected mounts the same authenticated routes on a router already
// wrapped with shared.RequireAuth.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	mount(r, d)
}

func mount(r chi.Router, d *shared.Deps) {
	if r == nil || d == nil {
		return
	}
	m := &module{d: d}
	r.Get("/discovery/friends", m.handleFriends)
	r.Get("/discovery/settings", m.handleGetSettings)
	r.Patch("/discovery/settings", m.handlePatchSettings)
	r.Post("/discovery/{userId}/suppress", m.handleSuppress)
	r.Delete("/discovery/{userId}/suppress", m.handleUnsuppress)
	r.Post("/connections", m.handleCreateConnection)
	r.Get("/connections", m.handleListConnections)
	r.Patch("/connections/{id}", m.handleReviewConnection)
	r.Delete("/connections/{id}", m.handleDeleteConnection)
}

// viewer returns the signed-in user from auth middleware context, falling
// back to the session cookie for routers that have no RequireAuth wrapper.
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
	if m.d.DB == nil {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Discovery is temporarily unavailable.")
		return 0, false
	}
	return id, true
}

func internalError(w http.ResponseWriter, d *shared.Deps, action string, err error) {
	if d != nil && d.Log != nil {
		d.Log.Error("discovery: "+action, "error", err)
	}
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError,
		"An unexpected error occurred.")
}

type fieldErrors map[string]string

func (f fieldErrors) add(field, message string) {
	if _, exists := f[field]; !exists {
		f[field] = message
	}
}

func writeValidation(w http.ResponseWriter, message string, details fieldErrors) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, message, map[string]string(details))
}

func parsePositiveID(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(chi.URLParam(r, param)), 10, 64)
	return id, err == nil && id > 0
}

func chunkIDs(ids []int64) [][]int64 {
	if len(ids) == 0 {
		return nil
	}
	chunks := make([][]int64, 0, (len(ids)+queryChunk-1)/queryChunk)
	for start := 0; start < len(ids); start += queryChunk {
		end := start + queryChunk
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func anyIDs(ids []int64) []any {
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

// userExists reports whether an active account with this id exists.
func (m *module) userExists(ctx context.Context, id int64) (bool, error) {
	var one int
	err := m.d.DB.QueryRowContext(ctx,
		`SELECT 1 FROM users WHERE id = ? AND is_active = 1 LIMIT 1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func blockedEitherWay(ctx context.Context, db *sql.DB, a, b int64) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx, `
		SELECT 1 FROM blocks
		WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)
		LIMIT 1`, a, b, b, a).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// loadUserSummaries resolves the exact R1 UserSummary shape for a set of ids
// in bounded chunks instead of one query per id.
func (m *module) loadUserSummaries(ctx context.Context, ids []int64) (map[int64]UserSummary, error) {
	result := make(map[int64]UserSummary, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	seen := make(map[int64]bool, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	for _, chunk := range chunkIDs(unique) {
		query := fmt.Sprintf(`
			SELECT u.id, COALESCE(u.username, ''), u.display_name, p.avatar_url
			FROM users u
			LEFT JOIN user_profiles p ON p.user_id = u.id
			WHERE u.id IN (%s)`, placeholders(len(chunk)))
		rows, err := m.d.DB.QueryContext(ctx, query, anyIDs(chunk)...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var summary UserSummary
			var avatar sql.NullString
			if err := rows.Scan(&summary.ID, &summary.Username, &summary.DisplayName, &avatar); err != nil {
				_ = rows.Close()
				return nil, err
			}
			summary.AvatarURL = nullString(avatar)
			result[summary.ID] = summary
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, err
		}
		_ = rows.Close()
	}
	return result, nil
}

// decodeFields reads a flat JSON object so both camelCase and snake_case keys
// can be accepted with explicit precedence rules.
func decodeFields(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	var raw map[string]json.RawMessage
	if !shared.DecodeJSON(w, r, &raw) {
		return nil, false
	}
	if raw == nil {
		raw = map[string]json.RawMessage{}
	}
	return raw, true
}

func decodeBool(raw json.RawMessage) (bool, bool) {
	if len(raw) == 0 {
		return false, false
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, false
	}
	return value, true
}

func decodeString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func decodeInt64(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}

// connectionInput accepts both the canonical camelCase body and the snake_case
// alias used by the frontend. When both keys are present, camelCase wins.
type connectionInput struct {
	UserID  int64
	Message string
}

func decodeConnectionInput(w http.ResponseWriter, r *http.Request) (connectionInput, bool) {
	var input connectionInput
	raw, ok := decodeFields(w, r)
	if !ok {
		return input, false
	}
	userRaw, hasCamel := raw["userId"]
	if !hasCamel {
		userRaw = raw["user_id"]
	}
	input.UserID, ok = decodeInt64(userRaw)
	if !ok {
		writeValidation(w, "Invalid connection request.", fieldErrors{
			"userId": "must be a positive integer user id",
		})
		return input, false
	}
	messageRaw := raw["message"]
	if len(messageRaw) > 0 {
		input.Message, ok = decodeString(messageRaw)
		if !ok {
			writeValidation(w, "Invalid connection request.", fieldErrors{
				"message": "must be a string",
			})
			return input, false
		}
	}
	input.Message = strings.TrimSpace(input.Message)
	if utf8.RuneCountInString(input.Message) > maxMessageRunes {
		writeValidation(w, "Invalid connection request.", fieldErrors{
			"message": fmt.Sprintf("must be at most %d characters", maxMessageRunes),
		})
		return input, false
	}
	return input, true
}
