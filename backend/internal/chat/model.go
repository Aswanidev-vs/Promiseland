package chat

import "database/sql"

// UserSummary is the shared R1 public-user shape, redefined locally because
// domain packages must not import each other.
type UserSummary struct {
	ID          int64   `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

// ChatServer is the exact R3 wire shape.
type ChatServer struct {
	ID          int64         `json:"id"`
	Owner       UserSummary   `json:"owner"`
	Name        string        `json:"name"`
	Slug        string        `json:"slug"`
	Description string        `json:"description"`
	IconURL     *string       `json:"icon_url"`
	MemberCount int           `json:"member_count"`
	Channels    []ChatChannel `json:"channels"`
	Role        *string       `json:"role"`
	CreatedAt   string        `json:"created_at"`
}

// ChatChannel is the exact R3 wire shape.
type ChatChannel struct {
	ID          int64        `json:"id"`
	ServerID    int64        `json:"server_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Type        string       `json:"type"`
	LastMessage *ChatMessage `json:"last_message"`
	UnreadCount int          `json:"unread_count"`
}

// Conversation is the exact R3 wire shape.
type Conversation struct {
	ID          int64         `json:"id"`
	Type        string        `json:"type"`
	Name        string        `json:"name"`
	Members     []UserSummary `json:"members"`
	LastMessage *ChatMessage  `json:"last_message"`
	UnreadCount int           `json:"unread_count"`
	CreatedAt   string        `json:"created_at"`
}

// MessageReaction is one aggregated reaction code for a message.
type MessageReaction struct {
	Reaction string `json:"reaction"`
	Count    int    `json:"count"`
	Me       bool   `json:"me"`
}

// ChatMessage is the exact R3 wire shape.
type ChatMessage struct {
	ID             int64             `json:"id"`
	ChannelID      *int64            `json:"channel_id"`
	ConversationID *int64            `json:"conversation_id"`
	Author         UserSummary       `json:"author"`
	ParentID       *int64            `json:"parent_id"`
	Body           string            `json:"body"`
	Reactions      []MessageReaction `json:"reactions"`
	CreatedAt      string            `json:"created_at"`
	EditedAt       string            `json:"edited_at"`
	Deleted        bool              `json:"deleted"`
}

// Report mirrors the R1 Report shape returned by /reports. Chat reuses the
// same reports table with target_type = 'message'.
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

type messageRow struct {
	ID             int64
	ChannelID      sql.NullInt64
	ConversationID sql.NullInt64
	AuthorID       int64
	ParentID       sql.NullInt64
	Body           string
	EditedAt       sql.NullString
	DeletedAt      sql.NullString
	CreatedAt      string
}

// scope identifies exactly one chat destination.
type scope struct {
	ChannelID      int64
	ConversationID int64
}

func (s scope) valid() bool {
	return (s.ChannelID > 0) != (s.ConversationID > 0)
}

func (s scope) channelArg() any {
	if s.ChannelID > 0 {
		return s.ChannelID
	}
	return nil
}

func (s scope) conversationArg() any {
	if s.ConversationID > 0 {
		return s.ConversationID
	}
	return nil
}

func (s scope) eventIDs() (int64, int64) {
	return s.ChannelID, s.ConversationID
}
