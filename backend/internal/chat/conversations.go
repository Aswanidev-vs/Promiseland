package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type conversationInput struct {
	MemberIDs []int64 `json:"memberIds"`
	// SnakeMemberIDs is the F1 frontend spelling of the same field.
	SnakeMemberIDs []int64 `json:"member_ids"`
	Name           string  `json:"name"`
}

// memberIDs accepts both the conventions' memberIds and the frontend's
// member_ids. The camelCase form wins when a client sends both.
func (in conversationInput) memberIDs() []int64 {
	if in.MemberIDs != nil {
		return in.MemberIDs
	}
	return in.SnakeMemberIDs
}

// pairKey is the canonical unordered key used to deduplicate direct chats.
func pairKey(a, b int64) string {
	if a > b {
		a, b = b, a
	}
	return strconv.FormatInt(a, 10) + ":" + strconv.FormatInt(b, 10)
}

func validateConversation(in conversationInput, viewerID int64) ([]int64, string, map[string]string) {
	details := map[string]string{}
	seen := map[int64]bool{}
	inputIDs := in.memberIDs()
	members := make([]int64, 0, len(inputIDs))
	for _, id := range inputIDs {
		if id <= 0 || id == viewerID {
			details["memberIds"] = "Member ids must be positive existing users other than yourself."
			continue
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		members = append(members, id)
	}
	if len(members) == 0 {
		details["memberIds"] = "Choose at least one other member."
	}
	if len(members) > maxGroupMembers-1 {
		details["memberIds"] = fmt.Sprintf("A group can have at most %d members including you.", maxGroupMembers)
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(in.Name)); n > 100 {
		details["name"] = "Name must be at most 100 characters."
	}
	return members, strings.TrimSpace(in.Name), details
}

func (m *module) usersExist(ctx context.Context, ids []int64) (bool, error) {
	if len(ids) == 0 {
		return false, nil
	}
	q := fmt.Sprintf(`SELECT COUNT(*) FROM users WHERE id IN (%s)`, placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	var count int
	if err := m.d.DB.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return false, err
	}
	return count == len(ids), nil
}

func (m *module) createConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	var in conversationInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	members, name, details := validateConversation(in, userID)
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}
	exists, err := m.usersExist(r.Context(), members)
	if err != nil {
		writeInternal(w, m, "chat: verify members", err)
		return
	}
	if !exists {
		writeValidation(w, map[string]string{"memberIds": "One or more selected users do not exist."})
		return
	}
	for _, memberID := range members {
		blocked, err := m.usersBlocked(r.Context(), userID, memberID)
		if err != nil {
			writeInternal(w, m, "chat: check blocks", err)
			return
		}
		if blocked {
			shared.WriteError(w, http.StatusForbidden, codeBlocked, "You cannot start a conversation with a blocked user.")
			return
		}
	}

	isDirect := len(members) == 1
	var key string
	if isDirect {
		key = pairKey(userID, members[0])
		var existingID int64
		err := m.d.DB.QueryRowContext(r.Context(),
			`SELECT id FROM conversations WHERE pair_key = ?`, key).Scan(&existingID)
		if err == nil {
			// Direct conversations are idempotent: reopening returns the
			// existing thread instead of creating a duplicate.
			_, _ = m.d.DB.ExecContext(r.Context(), `
				INSERT OR IGNORE INTO conversation_members(conversation_id, user_id, joined_at)
				VALUES(?,?,datetime('now'))`, existingID, userID)
			conversation, err := m.loadConversation(r.Context(), existingID, userID)
			if err != nil {
				writeInternal(w, m, "chat: load existing direct conversation", err)
				return
			}
			shared.WriteJSON(w, http.StatusOK, conversation)
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			writeInternal(w, m, "chat: lookup direct conversation", err)
			return
		}
	} else if name == "" {
		name = "Group chat"
	}

	conversationType := "group"
	if isDirect {
		conversationType = "direct"
		name = ""
	}
	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeInternal(w, m, "chat: begin create conversation", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var nullableKey any
	if isDirect {
		nullableKey = key
	}
	result, err := tx.ExecContext(r.Context(), `
		INSERT INTO conversations(type, name, created_by, pair_key, created_at)
		VALUES(?,?,?,?,datetime('now'))`, conversationType, name, userID, nullableKey)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			// Another request won the race for this pair; return that thread.
			var existingID int64
			if m.d.DB.QueryRowContext(r.Context(),
				`SELECT id FROM conversations WHERE pair_key = ?`, key).Scan(&existingID) == nil {
				conversation, loadErr := m.loadConversation(r.Context(), existingID, userID)
				if loadErr == nil {
					shared.WriteJSON(w, http.StatusOK, conversation)
					return
				}
			}
		}
		writeInternal(w, m, "chat: insert conversation", err)
		return
	}
	conversationID, _ := result.LastInsertId()
	allMembers := append([]int64{userID}, members...)
	for _, memberID := range allMembers {
		if _, err := tx.ExecContext(r.Context(), `
			INSERT INTO conversation_members(conversation_id, user_id, joined_at)
			VALUES(?,?,datetime('now'))`, conversationID, memberID); err != nil {
			writeInternal(w, m, "chat: insert conversation member", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeInternal(w, m, "chat: commit create conversation", err)
		return
	}
	conversation, err := m.loadConversation(r.Context(), conversationID, userID)
	if err != nil {
		writeInternal(w, m, "chat: load created conversation", err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, conversation)
}

func (m *module) listConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	rows, err := m.d.DB.QueryContext(r.Context(), `
		SELECT c.id FROM conversations c
		JOIN conversation_members cm ON cm.conversation_id = c.id
		WHERE cm.user_id = ?
		ORDER BY c.id DESC LIMIT ?`, userID, shared.MaxLimit)
	if err != nil {
		writeInternal(w, m, "chat: list conversations", err)
		return
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	result := make([]Conversation, 0, len(ids))
	for _, id := range ids {
		allowed, err := m.conversationAccess(r.Context(), id, userID)
		if err != nil || !allowed {
			continue
		}
		conversation, err := m.loadConversation(r.Context(), id, userID)
		if err == nil {
			result = append(result, conversation)
		}
	}
	shared.WriteJSON(w, http.StatusOK, result)
}
