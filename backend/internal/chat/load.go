package chat

import (
	"context"
	"database/sql"
	"fmt"
)

const messageSelect = `
SELECT m.id, m.channel_id, m.conversation_id, m.author_id, m.parent_message_id,
       m.body, m.edited_at, m.deleted_at, m.created_at
FROM messages m`

func scanMessageRows(rows *sql.Rows) ([]messageRow, error) {
	out := []messageRow{}
	for rows.Next() {
		var row messageRow
		if err := rows.Scan(&row.ID, &row.ChannelID, &row.ConversationID, &row.AuthorID,
			&row.ParentID, &row.Body, &row.EditedAt, &row.DeletedAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r messageRow) scope() scope {
	return scope{ChannelID: r.ChannelID.Int64, ConversationID: r.ConversationID.Int64}
}

func (m *module) messageRowByID(ctx context.Context, id int64) (messageRow, error) {
	var row messageRow
	err := m.d.DB.QueryRowContext(ctx, messageSelect+` WHERE m.id = ?`, id).Scan(
		&row.ID, &row.ChannelID, &row.ConversationID, &row.AuthorID,
		&row.ParentID, &row.Body, &row.EditedAt, &row.DeletedAt, &row.CreatedAt)
	return row, err
}

// loadUsers expands user IDs into the shared public summary shape.
func (m *module) loadUsers(ctx context.Context, ids []int64) map[int64]UserSummary {
	out := make(map[int64]UserSummary, len(ids))
	if len(ids) == 0 || m == nil || m.d == nil || m.d.DB == nil {
		return out
	}
	q := fmt.Sprintf(`
		SELECT u.id, COALESCE(%s, u.display_name, ''), u.display_name, p.avatar_url
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE u.id IN (%s)`, usernameExpr(m.d), placeholders(len(ids)))
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := m.d.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var u UserSummary
		var avatar sql.NullString
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &avatar); err != nil {
			continue
		}
		u.AvatarURL = nullString(avatar)
		out[u.ID] = u
	}
	return out
}

// loadMessages builds the exact R3 ChatMessage shape for a set of rows, with
// aggregated reactions and the viewer's own-reaction flag.
func (m *module) loadMessages(ctx context.Context, rows []messageRow, viewerID int64) []ChatMessage {
	out := make([]ChatMessage, 0, len(rows))
	if len(rows) == 0 {
		return out
	}
	authorIDs := make([]int64, 0, len(rows))
	messageIDs := make([]int64, 0, len(rows))
	for _, row := range rows {
		authorIDs = append(authorIDs, row.AuthorID)
		messageIDs = append(messageIDs, row.ID)
	}
	authors := m.loadUsers(ctx, authorIDs)
	reactions := map[int64][]MessageReaction{}
	q := fmt.Sprintf(`
		SELECT mr.message_id, mr.reaction, COUNT(*),
		       MAX(CASE WHEN mr.user_id = ? THEN 1 ELSE 0 END)
		FROM message_reactions mr
		WHERE mr.message_id IN (%s)
		GROUP BY mr.message_id, mr.reaction
		ORDER BY mr.message_id, mr.reaction`, placeholders(len(messageIDs)))
	args := append([]any{viewerID}, make([]any, len(messageIDs))...)
	for i, id := range messageIDs {
		args[i+1] = id
	}
	reactionRows, err := m.d.DB.QueryContext(ctx, q, args...)
	if err == nil {
		for reactionRows.Next() {
			var messageID int64
			var reaction MessageReaction
			var mine int
			if err := reactionRows.Scan(&messageID, &reaction.Reaction, &reaction.Count, &mine); err == nil {
				reaction.Me = mine == 1
				reactions[messageID] = append(reactions[messageID], reaction)
			}
		}
		reactionRows.Close()
	} else {
		m.logError("chat: load reactions failed", err)
	}

	for _, row := range rows {
		author := authors[row.AuthorID]
		if author.ID == 0 {
			author = UserSummary{ID: row.AuthorID, DisplayName: "unknown"}
		}
		list := reactions[row.ID]
		if list == nil {
			list = []MessageReaction{}
		}
		out = append(out, ChatMessage{
			ID:             row.ID,
			ChannelID:      nullInt64(row.ChannelID),
			ConversationID: nullInt64(row.ConversationID),
			Author:         author,
			ParentID:       nullInt64(row.ParentID),
			Body:           row.Body,
			Reactions:      list,
			CreatedAt:      row.CreatedAt,
			EditedAt:       row.EditedAt.String,
			Deleted:        row.DeletedAt.Valid,
		})
	}
	return out
}

func (m *module) loadMessage(ctx context.Context, id, viewerID int64) (ChatMessage, error) {
	row, err := m.messageRowByID(ctx, id)
	if err != nil {
		return ChatMessage{}, err
	}
	items := m.loadMessages(ctx, []messageRow{row}, viewerID)
	if len(items) == 0 {
		return ChatMessage{}, sql.ErrNoRows
	}
	return items[0], nil
}

// latestMessage returns the newest non-deleted message in a scope that the
// viewer is allowed to see, or nil when the scope has no visible history.
func (m *module) latestMessage(ctx context.Context, s scope, viewerID int64) *ChatMessage {
	clause, args := blockedAuthorClause(viewerID)
	query := messageSelect + `
		WHERE m.deleted_at IS NULL AND m.channel_id IS ? AND m.conversation_id IS ?` + clause +
		` ORDER BY m.id DESC LIMIT 1`
	params := append([]any{s.channelArg(), s.conversationArg()}, args...)
	rows, err := m.d.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil
	}
	found, err := scanMessageRows(rows)
	rows.Close()
	if err != nil || len(found) == 0 {
		return nil
	}
	messages := m.loadMessages(ctx, found, viewerID)
	if len(messages) == 0 {
		return nil
	}
	return &messages[0]
}

func (m *module) loadChannels(ctx context.Context, serverID, viewerID int64) []ChatChannel {
	channels := []ChatChannel{}
	if m == nil || m.d == nil || m.d.DB == nil {
		return channels
	}
	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT id, server_id, name, description, type FROM chat_channels
		WHERE server_id = ? ORDER BY id`, serverID)
	if err != nil {
		return channels
	}
	type channelRow struct {
		id, serverID           int64
		name, description, typ string
	}
	found := []channelRow{}
	for rows.Next() {
		var c channelRow
		if err := rows.Scan(&c.id, &c.serverID, &c.name, &c.description, &c.typ); err == nil {
			found = append(found, c)
		}
	}
	rows.Close()
	for _, c := range found {
		s := scope{ChannelID: c.id}
		channels = append(channels, ChatChannel{
			ID:          c.id,
			ServerID:    c.serverID,
			Name:        c.name,
			Description: c.description,
			Type:        c.typ,
			LastMessage: m.latestMessage(ctx, s, viewerID),
			UnreadCount: m.unreadCount(ctx, viewerID, s),
		})
	}
	return channels
}

func (m *module) loadServer(ctx context.Context, id, viewerID int64) (ChatServer, error) {
	var server ChatServer
	var owner UserSummary
	var ownerID int64
	var icon sql.NullString
	var role sql.NullString
	var avatar sql.NullString
	err := m.d.DB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT s.id, s.name, s.slug, s.description, s.icon_url, s.created_at, s.owner_id,
		       (SELECT COUNT(*) FROM chat_server_members sm WHERE sm.server_id = s.id),
		       (SELECT sm.role FROM chat_server_members sm WHERE sm.server_id = s.id AND sm.user_id = ?),
		       COALESCE(%s, u.display_name, ''), u.display_name, p.avatar_url
		FROM chat_servers s
		JOIN users u ON u.id = s.owner_id
		LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE s.id = ?`, usernameExpr(m.d)), viewerID, id).Scan(
		&server.ID, &server.Name, &server.Slug, &server.Description, &icon,
		&server.CreatedAt, &ownerID, &server.MemberCount, &role,
		&owner.Username, &owner.DisplayName, &avatar)
	if err != nil {
		return server, err
	}
	server.IconURL = nullString(icon)
	server.Channels = m.loadChannels(ctx, id, viewerID)
	if role.Valid {
		v := role.String
		server.Role = &v
	} else if ownerID == viewerID {
		v := "owner"
		server.Role = &v
	}
	owner.ID = ownerID
	owner.AvatarURL = nullString(avatar)
	server.Owner = owner
	return server, nil
}

func (m *module) conversationMemberIDs(ctx context.Context, conversationID int64) ([]int64, error) {
	rows, err := m.d.DB.QueryContext(ctx, `
		SELECT user_id FROM conversation_members
		WHERE conversation_id = ? ORDER BY user_id`, conversationID)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	err = rows.Err()
	rows.Close()
	return ids, err
}

func (m *module) loadConversation(ctx context.Context, id, viewerID int64) (Conversation, error) {
	var conversation Conversation
	var createdBy int64
	err := m.d.DB.QueryRowContext(ctx, `
		SELECT id, type, name, created_by, created_at FROM conversations WHERE id = ?`, id).
		Scan(&conversation.ID, &conversation.Type, &conversation.Name, &createdBy, &conversation.CreatedAt)
	if err != nil {
		return conversation, err
	}
	ids, err := m.conversationMemberIDs(ctx, id)
	if err != nil {
		return conversation, err
	}
	conversation.Members = make([]UserSummary, 0, len(ids))
	users := m.loadUsers(ctx, ids)
	for _, memberID := range ids {
		if user, ok := users[memberID]; ok {
			conversation.Members = append(conversation.Members, user)
		}
	}
	s := scope{ConversationID: id}
	conversation.LastMessage = m.latestMessage(ctx, s, viewerID)
	conversation.UnreadCount = m.unreadCount(ctx, viewerID, s)
	return conversation, nil
}
