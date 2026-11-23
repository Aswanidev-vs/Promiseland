package chat

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type messageInput struct {
	Body     string `json:"body"`
	ParentID *int64 `json:"parentId"`
	// SnakeParentID is the F1 frontend spelling of parentId.
	SnakeParentID *int64 `json:"parent_id"`
}

func (in messageInput) parentID() *int64 {
	if in.ParentID != nil {
		return in.ParentID
	}
	return in.SnakeParentID
}

type reactionInput struct {
	Reaction string `json:"reaction"`
}

type reportInput struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

// resolveScope resolves a {id} path parameter into a channel or conversation
// scope and enforces existence plus membership (and DM block rules).
func (m *module) resolveScope(w http.ResponseWriter, r *http.Request, kind string, userID int64) (scope, bool) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid id.")
		return scope{}, false
	}
	s := scope{}
	if kind == "channel" {
		s.ChannelID = id
	} else {
		s.ConversationID = id
	}
	var exists int
	table := "chat_channels"
	if kind != "channel" {
		table = "conversations"
	}
	if err := m.d.DB.QueryRowContext(r.Context(),
		"SELECT 1 FROM "+table+" WHERE id = ?", id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Scope not found.")
		} else {
			writeInternal(w, m, "chat: lookup scope", err)
		}
		return scope{}, false
	}
	allowed, err := m.canAccess(r.Context(), userID, s.ChannelID, s.ConversationID)
	if err != nil {
		writeInternal(w, m, "chat: scope access check", err)
		return scope{}, false
	}
	if !allowed {
		writeForbidden(w, "You cannot access this channel or conversation.")
		return scope{}, false
	}
	return s, true
}

func (m *module) listMessages(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := m.requireUser(w, r)
		if !ok {
			return
		}
		s, ok := m.resolveScope(w, r, kind, userID)
		if !ok {
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
		clause, args := blockedAuthorClause(userID)
		query := messageSelect + `
			WHERE m.channel_id IS ? AND m.conversation_id IS ? AND m.id < ?` + clause +
			` ORDER BY m.id DESC LIMIT ?`
		params := append([]any{s.channelArg(), s.conversationArg(), cursor}, args...)
		params = append(params, p.Limit+1)
		rows, err := m.d.DB.QueryContext(r.Context(), query, params...)
		if err != nil {
			writeInternal(w, m, "chat: list messages", err)
			return
		}
		found, err := scanMessageRows(rows)
		rows.Close()
		if err != nil {
			writeInternal(w, m, "chat: scan messages", err)
			return
		}
		page := shared.NewPage(m.loadMessages(r.Context(), found, userID))
		if len(found) > p.Limit {
			page.Items = page.Items[:p.Limit]
			page.NextCursor = shared.EncodeIDCursor(found[p.Limit-1].ID)
		}
		// Reading history marks the scope as read for the unread counters.
		m.markRead(r.Context(), userID, s)
		shared.WriteJSON(w, http.StatusOK, page)
	}
}

// parentDepth counts the ancestors of a message. A new reply is allowed only
// when its parent has at most two ancestors, giving a maximum depth of 2.
func (m *module) parentDepth(ctx context.Context, row messageRow) (int, error) {
	depth := 0
	current := row
	for current.ParentID.Valid && depth < 8 {
		parent, err := m.messageRowByID(ctx, current.ParentID.Int64)
		if err != nil {
			break
		}
		depth++
		current = parent
	}
	return depth, nil
}

func (m *module) eventRecipients(ctx context.Context, s scope, excludeID int64) []int64 {
	if s.ConversationID <= 0 {
		return nil
	}
	ids, err := m.conversationMemberIDs(ctx, s.ConversationID)
	if err != nil {
		return nil
	}
	recipients := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id != excludeID {
			recipients = append(recipients, id)
		}
	}
	return recipients
}

func (m *module) createMessage(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := m.requireUser(w, r)
		if !ok {
			return
		}
		s, ok := m.resolveScope(w, r, kind, userID)
		if !ok {
			return
		}
		var in messageInput
		if !shared.DecodeJSON(w, r, &in) {
			return
		}
		body := strings.TrimSpace(in.Body)
		n := utf8.RuneCountInString(body)
		if n < 1 || n > maxMessageBodyLen {
			writeValidation(w, map[string]string{"body": "Message body must be between 1 and 4000 characters."})
			return
		}
		if !m.messages.allow(userID, nowFn()) {
			shared.WriteError(w, http.StatusTooManyRequests, codeRateLimited, "You are sending messages too quickly. Slow down and try again.")
			return
		}
		var parentArg any
		if parentID := in.parentID(); parentID != nil && *parentID > 0 {
			parent, err := m.messageRowByID(r.Context(), *parentID)
			if err != nil {
				writeValidation(w, map[string]string{"parentId": "That message does not exist."})
				return
			}
			parentScope := parent.scope()
			if parentScope != s || parent.DeletedAt.Valid {
				writeValidation(w, map[string]string{"parentId": "Replies must reference a message in this conversation."})
				return
			}
			depth, err := m.parentDepth(r.Context(), parent)
			if err != nil {
				writeInternal(w, m, "chat: reply depth", err)
				return
			}
			if depth >= 2 {
				writeValidation(w, map[string]string{"parentId": "Replies may be at most two levels deep."})
				return
			}
			parentArg = *parentID
		}
		result, err := m.d.DB.ExecContext(r.Context(), `
			INSERT INTO messages(channel_id, conversation_id, author_id, parent_message_id, body, created_at)
			VALUES(?,?,?,?,?,datetime('now'))`,
			s.channelArg(), s.conversationArg(), userID, parentArg, body)
		if err != nil {
			writeInternal(w, m, "chat: insert message", err)
			return
		}
		messageID, _ := result.LastInsertId()
		message, err := m.loadMessage(r.Context(), messageID, userID)
		if err != nil {
			writeInternal(w, m, "chat: load created message", err)
			return
		}
		channelID, conversationID := s.eventIDs()
		m.publish("message.new", message, channelID, conversationID, m.eventRecipients(r.Context(), s, userID))
		if s.ConversationID > 0 {
			ids, err := m.conversationMemberIDs(r.Context(), s.ConversationID)
			if err == nil {
				for _, id := range ids {
					// Optimistic: a missing notifications table must not fail delivery.
					m.notifyUser(id, userID, "chat_message", "conversation", s.ConversationID)
				}
			}
		}
		shared.WriteJSON(w, http.StatusCreated, message)
	}
}

func (m *module) messageWithAccess(w http.ResponseWriter, r *http.Request, userID int64) (messageRow, bool) {
	id, err := parseIDParam(r, "id")
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, codeInvalidID, "Invalid message id.")
		return messageRow{}, false
	}
	row, err := m.messageRowByID(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Message not found.")
		return messageRow{}, false
	}
	if err != nil {
		writeInternal(w, m, "chat: load message", err)
		return messageRow{}, false
	}
	allowed, err := m.canAccess(r.Context(), userID, row.ChannelID.Int64, row.ConversationID.Int64)
	if err != nil {
		writeInternal(w, m, "chat: message access check", err)
		return messageRow{}, false
	}
	if !allowed {
		writeForbidden(w, "You cannot access this message.")
		return messageRow{}, false
	}
	return row, true
}

func (m *module) reactToMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	row, ok := m.messageWithAccess(w, r, userID)
	if !ok {
		return
	}
	var in reactionInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	reaction := strings.ToLower(strings.TrimSpace(in.Reaction))
	if !validReactions[reaction] {
		writeValidation(w, map[string]string{"reaction": "Reaction must be like, heart, star or laugh."})
		return
	}
	tx, err := m.d.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeInternal(w, m, "chat: begin reaction", err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var existing int
	err = tx.QueryRowContext(r.Context(), `
		SELECT 1 FROM message_reactions
		WHERE message_id = ? AND user_id = ? AND reaction = ?`, row.ID, userID, reaction).Scan(&existing)
	if err == nil {
		_, err = tx.ExecContext(r.Context(), `
			DELETE FROM message_reactions
			WHERE message_id = ? AND user_id = ? AND reaction = ?`, row.ID, userID, reaction)
	} else if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.ExecContext(r.Context(), `
			INSERT OR IGNORE INTO message_reactions(message_id, user_id, reaction, created_at)
			VALUES(?,?,?,datetime('now'))`, row.ID, userID, reaction)
	}
	if err != nil {
		writeInternal(w, m, "chat: toggle reaction", err)
		return
	}
	if err := tx.Commit(); err != nil {
		writeInternal(w, m, "chat: commit reaction", err)
		return
	}
	message, err := m.loadMessage(r.Context(), row.ID, userID)
	if err != nil {
		writeInternal(w, m, "chat: reload reacted message", err)
		return
	}
	s := row.scope()
	channelID, conversationID := s.eventIDs()
	m.publish("reaction.updated", message, channelID, conversationID, m.eventRecipients(r.Context(), s, userID))
	shared.WriteJSON(w, http.StatusOK, message)
}

func (m *module) deleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	row, ok := m.messageWithAccess(w, r, userID)
	if !ok {
		return
	}
	allowed := row.AuthorID == userID || m.isPlatformModerator(r.Context(), userID)
	if !allowed && row.ChannelID.Valid {
		serverID, err := m.serverIDForChannel(r.Context(), row.ChannelID.Int64)
		if err == nil {
			role, err := m.serverRole(r.Context(), serverID, userID)
			if err == nil && (role == "owner" || role == "moderator") {
				allowed = true
			}
		}
	}
	if !allowed {
		writeForbidden(w, "Only the author or a moderator can delete this message.")
		return
	}
	if _, err := m.d.DB.ExecContext(r.Context(), `
		UPDATE messages SET deleted_at = datetime('now') WHERE id = ? AND deleted_at IS NULL`, row.ID); err != nil {
		writeInternal(w, m, "chat: soft delete message", err)
		return
	}
	message, err := m.loadMessage(r.Context(), row.ID, userID)
	if err != nil {
		writeInternal(w, m, "chat: reload deleted message", err)
		return
	}
	s := row.scope()
	channelID, conversationID := s.eventIDs()
	m.publish("message.deleted", message, channelID, conversationID, m.eventRecipients(r.Context(), s, userID))
	shared.WriteJSON(w, http.StatusOK, nil)
}

func (m *module) reportMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	row, ok := m.messageWithAccess(w, r, userID)
	if !ok {
		return
	}
	var in reportInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	reason := strings.ToLower(strings.TrimSpace(in.Reason))
	details := map[string]string{}
	if !validReportReasons[reason] {
		details["reason"] = "Choose a supported report reason."
	}
	if utf8.RuneCountInString(in.Details) > 2000 {
		details["details"] = "Details must be at most 2000 characters."
	}
	if len(details) > 0 {
		writeValidation(w, details)
		return
	}
	if !tableExists(m.d, "reports") {
		shared.WriteError(w, http.StatusServiceUnavailable, shared.CodeDBUnavailable, "Reporting is temporarily unavailable.")
		return
	}
	var existing int
	err := m.d.DB.QueryRowContext(r.Context(), `
		SELECT 1 FROM reports
		WHERE reporter_id = ? AND target_type = 'message' AND target_id = ? AND status = 'open'
		LIMIT 1`, userID, row.ID).Scan(&existing)
	if err == nil {
		shared.WriteError(w, http.StatusConflict, codeAlreadyReport, "You have already reported this message.")
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeInternal(w, m, "chat: duplicate report check", err)
		return
	}
	var detailsValue any
	if strings.TrimSpace(in.Details) != "" {
		detailsValue = strings.TrimSpace(in.Details)
	}
	result, err := m.d.DB.ExecContext(r.Context(), `
		INSERT INTO reports(reporter_id, target_type, target_id, reason, details, status, created_at)
		VALUES(?, 'message', ?, ?, ?, 'open', datetime('now'))`, userID, row.ID, reason, detailsValue)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			shared.WriteError(w, http.StatusConflict, codeAlreadyReport, "You have already reported this message.")
		} else {
			writeInternal(w, m, "chat: insert report", err)
		}
		return
	}
	reportID, _ := result.LastInsertId()
	report := Report{ID: reportID, ReporterID: userID, TargetType: "message", TargetID: row.ID, Reason: reason, Status: "open"}
	var storedDetails sql.NullString
	if err := m.d.DB.QueryRowContext(r.Context(), `
		SELECT created_at, details, status FROM reports WHERE id = ?`, reportID).
		Scan(&report.CreatedAt, &storedDetails, &report.Status); err != nil {
		writeInternal(w, m, "chat: reload report", err)
		return
	}
	report.Details = nullString(storedDetails)
	shared.WriteJSON(w, http.StatusCreated, report)
}
