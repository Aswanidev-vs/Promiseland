package adult

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

const matchQueryTimeout = 8 * time.Second

type matchRow struct {
	ID        int64
	UserA     int64
	UserB     int64
	CreatedAt string
}

// handleMatches lists only ACTIVE matches for the viewer. There is no global
// discovery feed: a profile is never returned to someone who is not matched.
func (m *module) handleMatches(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	pg := shared.ParsePagination(r)
	var afterID int64
	if pg.Cursor != "" {
		id, err := shared.DecodeIDCursor(pg.Cursor)
		if err != nil {
			writeValidation(w, map[string]string{"cursor": "must be a valid cursor"})
			return
		}
		afterID = id
	}
	if afterID == 0 {
		afterID = math.MaxInt64
	}

	ctx, cancel := context.WithTimeout(r.Context(), matchQueryTimeout)
	defer cancel()

	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT id, user_a, user_b, created_at FROM adult_matches
		WHERE (user_a = ? OR user_b = ?) AND unmatched_at IS NULL AND id < ?
		ORDER BY id DESC LIMIT ?`, userID, userID, afterID, pg.Limit+1)
	if err != nil {
		internalError(w, m.d, "list adult matches", err)
		return
	}
	matches := []matchRow{}
	for rows.Next() {
		var row matchRow
		if err := rows.Scan(&row.ID, &row.UserA, &row.UserB, &row.CreatedAt); err != nil {
			_ = rows.Close()
			internalError(w, m.d, "scan adult match", err)
			return
		}
		matches = append(matches, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		internalError(w, m.d, "iterate adult matches", err)
		return
	}
	_ = rows.Close()

	pageResult := page[AdultMatch]{Items: []AdultMatch{}, NextCursor: ""}
	if len(matches) > pg.Limit {
		pageResult.NextCursor = shared.EncodeIDCursor(matches[pg.Limit-1].ID)
		matches = matches[:pg.Limit]
	}
	if len(matches) == 0 {
		shared.WriteJSON(w, http.StatusOK, pageResult)
		return
	}

	ids := make([]int64, 0, len(matches)*2)
	for _, row := range matches {
		ids = append(ids, row.UserA, row.UserB)
	}
	summaries, err := m.loadUserSummaries(ctx, ids)
	if err != nil {
		internalError(w, m.d, "load adult match users", err)
		return
	}
	conversations, err := m.conversationIDs(ctx, matches)
	if err != nil {
		internalError(w, m.d, "load adult match conversations", err)
		return
	}

	for _, row := range matches {
		a, okA := summaries[row.UserA]
		b, okB := summaries[row.UserB]
		if !okA || !okB {
			continue // accounts deleted after the match; FK cleanup is pending
		}
		item := AdultMatch{
			ID:        row.ID,
			Users:     []UserSummary{a, b},
			CreatedAt: row.CreatedAt,
		}
		if id, exists := conversations[pairKey(row.UserA, row.UserB)]; exists {
			value := id
			item.ConversationID = &value
		}
		pageResult.Items = append(pageResult.Items, item)
	}
	shared.WriteJSON(w, http.StatusOK, pageResult)
}

// handleUnmatch lets either party end a match immediately. Like rows are
// removed too, so re-matching always requires two fresh positive actions.
func (m *module) handleUnmatch(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.authorize(w, r)
	if !ok {
		return
	}
	matchID, valid := positiveURLID(r, "id")
	if !valid {
		writeValidation(w, map[string]string{"id": "must be a positive integer match id"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), matchQueryTimeout)
	defer cancel()

	var userA, userB int64
	var unmatched sql.NullString
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT user_a, user_b, unmatched_at FROM adult_matches WHERE id = ?`, matchID).
		Scan(&userA, &userB, &unmatched)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Match not found.")
		return
	}
	if err != nil {
		internalError(w, m.d, "load adult match", err)
		return
	}
	if userID != userA && userID != userB {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Match not found.")
		return
	}
	if unmatched.Valid && strings.TrimSpace(unmatched.String) != "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	tx, err := m.d.DB.BeginTx(ctx, nil)
	if err != nil {
		internalError(w, m.d, "begin adult unmatch", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE adult_matches SET unmatched_at = datetime('now')
		WHERE id = ? AND unmatched_at IS NULL`, matchID); err != nil {
		internalError(w, m.d, "unmatch", err)
		return
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM adult_likes
		WHERE (actor_id = ? AND target_id = ?) OR (actor_id = ? AND target_id = ?)`,
		userA, userB, userB, userA); err != nil {
		internalError(w, m.d, "clear likes after unmatch", err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(w, m.d, "commit adult unmatch", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *module) loadUserSummaries(ctx context.Context, ids []int64) (map[int64]UserSummary, error) {
	result := map[int64]UserSummary{}
	seen := map[int64]bool{}
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(unique)), ",")
	args := make([]any, len(unique))
	for i, id := range unique {
		args[i] = id
	}
	// Email, location and other private account fields are deliberately absent.
	query := fmt.Sprintf(`
		SELECT u.id, COALESCE(u.username, ''), u.display_name, p.avatar_url
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id IN (%s)`, placeholders)
	rows, err := m.d.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var summary UserSummary
		var avatar sql.NullString
		if err := rows.Scan(&summary.ID, &summary.Username, &summary.DisplayName, &avatar); err != nil {
			return nil, err
		}
		if avatar.Valid {
			value := avatar.String
			summary.AvatarURL = &value
		}
		result[summary.ID] = summary
	}
	return result, rows.Err()
}

// conversationIDs resolves conversation_id from the existing R3 table by the
// stable pair key; it never writes an adult-specific column into chat tables.
func (m *module) conversationIDs(ctx context.Context, matches []matchRow) (map[string]int64, error) {
	result := map[string]int64{}
	if len(matches) == 0 || !loadConversationSchema(ctx, m.d.DB).usable() {
		return result, nil
	}
	keys := make([]string, 0, len(matches))
	for _, row := range matches {
		keys = append(keys, pairKey(row.UserA, row.UserB))
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(keys)), ",")
	args := make([]any, len(keys))
	for i, key := range keys {
		args[i] = key
	}
	rows, err := m.d.DB.QueryContext(ctx,
		fmt.Sprintf(`SELECT pair_key, id FROM conversations WHERE pair_key IN (%s)`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		var id int64
		if err := rows.Scan(&key, &id); err != nil {
			return nil, err
		}
		result[key] = id
	}
	return result, rows.Err()
}
