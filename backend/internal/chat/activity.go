package chat

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
)

type typingInput struct {
	ChannelID      *int64 `json:"channelId"`
	ConversationID *int64 `json:"conversationId"`
	IsTyping       *bool  `json:"isTyping"`

	// F1 frontend spellings; the camelCase forms above win when both are sent.
	SnakeChannelID      *int64 `json:"channel_id"`
	SnakeConversationID *int64 `json:"conversation_id"`
	SnakeIsTyping       *bool  `json:"is_typing"`
}

func (in typingInput) channelID() *int64 {
	if in.ChannelID != nil {
		return in.ChannelID
	}
	return in.SnakeChannelID
}

func (in typingInput) conversationID() *int64 {
	if in.ConversationID != nil {
		return in.ConversationID
	}
	return in.SnakeConversationID
}

func (in typingInput) isTyping() bool {
	if in.IsTyping != nil {
		return *in.IsTyping
	}
	if in.SnakeIsTyping != nil {
		return *in.SnakeIsTyping
	}
	return false
}

func (m *module) sendTyping(w http.ResponseWriter, r *http.Request) {
	userID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	var in typingInput
	if !shared.DecodeJSON(w, r, &in) {
		return
	}
	channelID, conversationID := int64(0), int64(0)
	if id := in.channelID(); id != nil && *id > 0 {
		channelID = *id
	}
	if id := in.conversationID(); id != nil && *id > 0 {
		conversationID = *id
	}
	if (channelID > 0) == (conversationID > 0) {
		writeValidation(w, map[string]string{"scope": "Provide exactly one of channelId or conversationId."})
		return
	}
	s := scope{ChannelID: channelID, ConversationID: conversationID}
	table := "chat_channels"
	targetID := channelID
	if conversationID > 0 {
		table = "conversations"
		targetID = conversationID
	}
	var exists int
	if err := m.d.DB.QueryRowContext(r.Context(),
		"SELECT 1 FROM "+table+" WHERE id = ?", targetID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			shared.WriteError(w, http.StatusNotFound, shared.CodeNotFound, "Scope not found.")
		} else {
			writeInternal(w, m, "chat: typing scope lookup", err)
		}
		return
	}
	allowed, err := m.canAccess(r.Context(), userID, channelID, conversationID)
	if err != nil {
		writeInternal(w, m, "chat: typing access check", err)
		return
	}
	if !allowed {
		writeForbidden(w, "You cannot access this channel or conversation.")
		return
	}
	if !m.typing.allow(userID, channelID, conversationID, nowFn()) {
		shared.WriteError(w, http.StatusTooManyRequests, codeRateLimited, "Typing updates are rate limited.")
		return
	}
	if _, err := m.d.DB.ExecContext(r.Context(), `
		DELETE FROM typing_state WHERE user_id = ? AND channel_id IS ? AND conversation_id IS ?`,
		userID, s.channelArg(), s.conversationArg()); err != nil {
		writeInternal(w, m, "chat: clear typing state", err)
		return
	}
	if in.isTyping() {
		if _, err := m.d.DB.ExecContext(r.Context(), `
			INSERT INTO typing_state(user_id, channel_id, conversation_id, updated_at)
			VALUES(?,?,?,datetime('now'))`, userID, s.channelArg(), s.conversationArg()); err != nil {
			writeInternal(w, m, "chat: store typing state", err)
			return
		}
	}
	payload := map[string]any{"user_id": userID, "is_typing": in.isTyping()}
	m.publish("typing", payload, channelID, conversationID, m.eventRecipients(r.Context(), s, userID))
	shared.WriteJSON(w, http.StatusOK, nil)
}

// getPresence returns a REST snapshot of the statuses pushed authoritatively
// over the WebSocket by the realtime hub.
func (m *module) getPresence(w http.ResponseWriter, r *http.Request) {
	viewerID, ok := m.requireUser(w, r)
	if !ok {
		return
	}
	raw := r.URL.Query().Get("ids")
	result := map[int64]string{}
	if strings.TrimSpace(raw) == "" {
		shared.WriteJSON(w, http.StatusOK, result)
		return
	}
	seen := map[int64]bool{}
	for _, part := range strings.Split(raw, ",") {
		if len(result) >= maxPresenceIDs {
			break
		}
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		status := "offline"
		if id == viewerID {
			status = "online"
		} else if tableExists(m.d, "chat_presence") {
			var stored string
			err := m.d.DB.QueryRowContext(r.Context(), `
				SELECT status FROM chat_presence
				WHERE user_id = ? AND updated_at >= datetime('now', '-2 minutes')`, id).Scan(&stored)
			if err == nil && (stored == "online" || stored == "away") {
				status = stored
			}
		}
		result[id] = status
	}
	shared.WriteJSON(w, http.StatusOK, result)
}
