// Package chat implements Release 3 community communication: chat servers,
// text channels, direct/group conversations, persistent messages, reactions,
// typing and presence. It publishes realtime events through internal/realtime.
package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Aswanidev-vs/promiseland/backend/internal/shared"
	"github.com/go-chi/chi/v5"
)

const (
	codeValidation     = "validation_error"
	codeForbidden      = "forbidden"
	codeSlugTaken      = "slug_taken"
	codeChannelExists  = "channel_exists"
	codeRateLimited    = "rate_limited"
	codeBlocked        = "blocked"
	codeAlreadyReport  = "already_reported"
	codeInvalidID      = "invalid_id"
	codeScopeRequired  = "scope_required"
	maxMessageBodyLen  = 4000
	maxGroupMembers    = 20
	maxPresenceIDs     = 100
	presenceFreshFor   = 2 * time.Minute
	presenceTouchEvery = 30 * time.Second
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{3,40}$`)

var validReactions = map[string]bool{
	"like": true, "heart": true, "star": true, "laugh": true,
}

var validReportReasons = map[string]bool{
	"spam": true, "harassment": true, "impersonation": true,
	"hate_abuse": true, "inappropriate": true, "copyright": true,
	"spoiler_abuse": true, "unsafe_event": true, "other": true,
}

type module struct {
	d             *shared.Deps
	messages      *messageLimiter
	typing        *typingLimiter
	presenceMu    sync.Mutex
	presenceTouch map[int64]time.Time
}

func newModule(d *shared.Deps) *module {
	return &module{
		d:             d,
		messages:      newMessageLimiter(),
		typing:        newTypingLimiter(),
		presenceTouch: make(map[int64]time.Time),
	}
}

// Register mounts public chat routes. All chat handlers still enforce
// authentication themselves via module.viewer, so this registration is safe
// on either a public router or one wrapped with shared.RequireAuth.
func Register(r chi.Router, d *shared.Deps) {
	if r == nil || d == nil {
		return
	}
	m := newModule(d)
	m.configureRealtime()
	r.Get("/chat/servers", m.listServers)
	r.Get("/chat/servers/{id}/channels", m.listChannels)
	r.Get("/chat/conversations", m.listConversations)
	r.Get("/chat/channels/{id}/messages", m.listMessages("channel"))
	r.Get("/chat/conversations/{id}/messages", m.listMessages("conversation"))
	r.Get("/chat/presence", m.getPresence)
}

// RegisterProtected mounts authenticated chat routes. It can be mounted on a
// chi router wrapped with shared.RequireAuth; handlers repeat the check.
func RegisterProtected(r chi.Router, d *shared.Deps) {
	if r == nil || d == nil {
		return
	}
	m := newModule(d)
	m.configureRealtime()
	r.Post("/chat/servers", m.createServer)
	r.Post("/chat/servers/{id}/join", m.joinServer)
	r.Post("/chat/servers/{id}/channels", m.createChannel)
	r.Post("/chat/conversations", m.createConversation)
	r.Post("/chat/channels/{id}/messages", m.createMessage("channel"))
	r.Post("/chat/conversations/{id}/messages", m.createMessage("conversation"))
	r.Post("/chat/messages/{id}/reactions", m.reactToMessage)
	r.Delete("/chat/messages/{id}", m.deleteMessage)
	r.Post("/chat/messages/{id}/report", m.reportMessage)
	r.Post("/chat/typing", m.sendTyping)
}

func (m *module) configureRealtime() {
	hub := realtimeHub()
	if hub == nil {
		return
	}
	hub.SetDeps(m.d)
	hub.SetAccessChecker(func(userID, channelID, conversationID int64) bool {
		if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
			return false
		}
		ok, _ := m.canAccess(context.Background(), userID, channelID, conversationID)
		return ok
	})
}

// viewer accepts either the auth middleware context or a live session cookie.
// This keeps both Register and RegisterProtected usable with the existing
// router, which only adds OptionalAuth on some subrouters.
func (m *module) viewer(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	if id, ok := shared.UserID(r); ok {
		return id, true
	}
	if m == nil || m.d == nil {
		return 0, false
	}
	return m.d.SessionUserID(r)
}

func (m *module) requireUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := m.viewer(r)
	if !ok {
		shared.WriteError(w, http.StatusUnauthorized, shared.CodeAuthRequired, "Sign in to continue.")
		return 0, false
	}
	m.touchPresence(userID)
	return userID, true
}

func (m *module) logError(message string, err error) {
	if m != nil && m.d != nil && m.d.Log != nil {
		m.d.Log.Warn(message, "error", err)
	}
}

func writeValidation(w http.ResponseWriter, details map[string]string) {
	shared.WriteErrorDetails(w, http.StatusBadRequest, codeValidation, "Please correct the highlighted fields.", details)
}

func writeForbidden(w http.ResponseWriter, message string) {
	shared.WriteError(w, http.StatusForbidden, codeForbidden, message)
}

func writeInternal(w http.ResponseWriter, m *module, message string, err error) {
	m.logError(message, err)
	shared.WriteError(w, http.StatusInternalServerError, shared.CodeInternalError, "The request could not be completed.")
}

func parseIDParam(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(chi.URLParam(r, key))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return id, nil
}

func tableExists(d *shared.Deps, name string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	var one int
	err := d.DB.QueryRow(`
		SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1`, name).Scan(&one)
	return err == nil
}

func columnExists(d *shared.Deps, table, column string) bool {
	if d == nil || d.DB == nil {
		return false
	}
	rows, err := d.DB.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			continue
		}
		if strings.EqualFold(name, column) {
			return true
		}
	}
	return false
}

func placeholders(n int) string {
	if n <= 0 {
		return "NULL"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func usernameExpr(d *shared.Deps) string {
	if columnExists(d, "users", "username") {
		return "NULLIF(u.username, '')"
	}
	return "NULLIF(u.display_name, '')"
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func nullInt64(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

// blockedAuthorClause hides messages written by anybody with a block relation
// to the viewer, in either direction. It is used for channel history where
// denying the whole channel would be too coarse.
func blockedAuthorClause(viewerID int64) (string, []any) {
	if viewerID <= 0 {
		return "", nil
	}
	return ` AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = m.author_id AND b.blocked_id = ?)` +
			` AND NOT EXISTS (SELECT 1 FROM blocks b WHERE b.blocker_id = ? AND b.blocked_id = m.author_id)`,
		[]any{viewerID, viewerID}
}

// usersBlocked reports whether either user has blocked the other.
func (m *module) usersBlocked(ctx context.Context, a, b int64) (bool, error) {
	if m == nil || m.d == nil || m.d.DB == nil || a <= 0 || b <= 0 || !tableExists(m.d, "blocks") {
		return false, nil
	}
	var one int
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM blocks
		WHERE (blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)
		LIMIT 1`, a, b, b, a).Scan(&one)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (m *module) serverRole(ctx context.Context, serverID, userID int64) (string, error) {
	if m == nil || m.d == nil || m.d.DB == nil || serverID <= 0 || userID <= 0 {
		return "", nil
	}
	var ownerID int64
	err := m.d.DB.QueryRowContext(ctx, `SELECT owner_id FROM chat_servers WHERE id = ?`, serverID).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if ownerID == userID {
		return "owner", nil
	}
	var role string
	err = m.d.DB.QueryRowContext(ctx, `
		SELECT role FROM chat_server_members WHERE server_id = ? AND user_id = ?`, serverID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return role, err
}

// channelAccess checks that the channel exists and the user is a server member.
func (m *module) channelAccess(ctx context.Context, channelID, userID int64) (bool, error) {
	if m == nil || m.d == nil || m.d.DB == nil || channelID <= 0 || userID <= 0 {
		return false, nil
	}
	var one int
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM chat_channels c
		JOIN chat_server_members m ON m.server_id = c.server_id
		WHERE c.id = ? AND m.user_id = ? LIMIT 1`, channelID, userID).Scan(&one)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

// conversationAccess checks membership and that the viewer has no block
// relation with any other member. Blocked users cannot read each other's DMs.
func (m *module) conversationAccess(ctx context.Context, conversationID, userID int64) (bool, error) {
	if m == nil || m.d == nil || m.d.DB == nil || conversationID <= 0 || userID <= 0 {
		return false, nil
	}
	var member int
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT 1 FROM conversation_members
		WHERE conversation_id = ? AND user_id = ? LIMIT 1`, conversationID, userID).Scan(&member)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	ids, err := m.conversationMemberIDs(ctx, conversationID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == userID {
			continue
		}
		blocked, err := m.usersBlocked(ctx, userID, id)
		if err != nil {
			return false, err
		}
		if blocked {
			return false, nil
		}
	}
	return true, nil
}

func (m *module) canAccess(ctx context.Context, userID, channelID, conversationID int64) (bool, error) {
	if channelID > 0 && conversationID == 0 {
		return m.channelAccess(ctx, channelID, userID)
	}
	if conversationID > 0 && channelID == 0 {
		return m.conversationAccess(ctx, conversationID, userID)
	}
	return false, nil
}

// isPlatformModerator checks both the dedicated grant table and the legacy
// boolean columns used by the R1 moderation module.
func (m *module) isPlatformModerator(ctx context.Context, userID int64) bool {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
		return false
	}
	if tableExists(m.d, "platform_moderators") {
		var one int
		err := m.d.DB.QueryRowContext(ctx, `SELECT 1 FROM platform_moderators WHERE user_id = ? LIMIT 1`, userID).Scan(&one)
		if err == nil {
			return true
		}
	}
	for _, column := range []string{"is_platform_moderator", "is_moderator"} {
		if !columnExists(m.d, "users", column) {
			continue
		}
		var enabled int
		err := m.d.DB.QueryRowContext(ctx,
			fmt.Sprintf(`SELECT COALESCE(%s, 0) FROM users WHERE id = ?`, column), userID).Scan(&enabled)
		if err == nil && enabled != 0 {
			return true
		}
	}
	return false
}

// notifyUser is deliberately optimistic: a missing notifications table or a
// schema mismatch must never fail the chat operation that triggered it.
func (m *module) notifyUser(userID, actorID int64, notificationType, targetType string, targetID int64) {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 || userID == actorID || !tableExists(m.d, "notifications") {
		return
	}
	_, err := m.d.DB.Exec(`
		INSERT INTO notifications(user_id, type, actor_id, target_type, target_id, created_at)
		VALUES(?,?,?,?,?,datetime('now'))`, userID, actorID, notificationType, targetType, targetID)
	if err != nil {
		m.logError("chat: notification insert skipped", err)
	}
}

// touchPresence throttles the heartbeat so read handlers do not write on every
// request. A status is considered fresh for presenceFreshFor.
func (m *module) touchPresence(userID int64) {
	if userID <= 0 || m == nil || m.d == nil || m.d.DB == nil || !tableExists(m.d, "chat_presence") {
		return
	}
	now := time.Now()
	m.presenceMu.Lock()
	if last, ok := m.presenceTouch[userID]; ok && now.Sub(last) < presenceTouchEvery {
		m.presenceMu.Unlock()
		return
	}
	m.presenceTouch[userID] = now
	m.presenceMu.Unlock()
	_, err := m.d.DB.Exec(`
		INSERT INTO chat_presence(user_id, status, updated_at) VALUES(?, 'online', datetime('now'))
		ON CONFLICT(user_id) DO UPDATE SET status = 'online', updated_at = datetime('now')`, userID)
	if err != nil {
		m.logError("chat: presence heartbeat skipped", err)
	}
}

// markRead advances the caller's read cursor for a scope. Viewing history is
// what marks a channel or conversation as read.
func (m *module) markRead(ctx context.Context, userID int64, s scope) {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 || !tableExists(m.d, "chat_reads") {
		return
	}
	_, err := m.d.DB.ExecContext(ctx, `
		DELETE FROM chat_reads WHERE user_id = ? AND channel_id IS ? AND conversation_id IS ?`,
		userID, s.channelArg(), s.conversationArg())
	if err != nil {
		m.logError("chat: clear read cursor failed", err)
		return
	}
	_, err = m.d.DB.ExecContext(ctx, `
		INSERT INTO chat_reads(user_id, channel_id, conversation_id, last_message_id, updated_at)
		SELECT ?, ?, ?, COALESCE(MAX(id), 0), datetime('now')
		FROM messages
		WHERE channel_id IS ? AND conversation_id IS ?`,
		userID, s.channelArg(), s.conversationArg(), s.channelArg(), s.conversationArg())
	if err != nil {
		m.logError("chat: advance read cursor failed", err)
	}
}

func (m *module) unreadCount(ctx context.Context, userID int64, s scope) int {
	if m == nil || m.d == nil || m.d.DB == nil || userID <= 0 {
		return 0
	}
	var last int64
	if tableExists(m.d, "chat_reads") {
		_ = m.d.DB.QueryRowContext(ctx, `
			SELECT last_message_id FROM chat_reads
			WHERE user_id = ? AND channel_id IS ? AND conversation_id IS ?`,
			userID, s.channelArg(), s.conversationArg()).Scan(&last)
	}
	clause, args := blockedAuthorClause(userID)
	query := `SELECT COUNT(*) FROM messages m
		WHERE m.deleted_at IS NULL AND m.id > ? AND m.author_id <> ?
		  AND m.channel_id IS ? AND m.conversation_id IS ?` + clause
	params := append([]any{last, userID, s.channelArg(), s.conversationArg()}, args...)
	var count int
	if err := m.d.DB.QueryRowContext(ctx, query, params...).Scan(&count); err != nil {
		return 0
	}
	return count
}
