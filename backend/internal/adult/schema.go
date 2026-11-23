package adult

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// conversationSchema caches the parts of the existing R3 chat schema that the
// match flow depends on. The adult package does not import the chat package;
// it inspects the schema locally so a different or absent chat schema degrades
// gracefully (a match is still created, with conversation_id = null).
type conversationSchema struct {
	conversations bool
	members       bool
}

func loadConversationSchema(ctx context.Context, db *sql.DB) conversationSchema {
	result := conversationSchema{}
	if db == nil {
		return result
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	result.conversations = hasColumns(checkCtx, db, "conversations",
		"type", "created_by", "pair_key", "created_at")
	result.members = hasColumns(checkCtx, db, "conversation_members",
		"conversation_id", "user_id", "joined_at")
	return result
}

func (s conversationSchema) usable() bool { return s.conversations && s.members }

func hasColumns(ctx context.Context, db *sql.DB, table string, columns ...string) bool {
	var exists int
	err := db.QueryRowContext(ctx,
		`SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ? LIMIT 1`, table).
		Scan(&exists)
	if err != nil {
		return false
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info("`+strings.ReplaceAll(table, `"`, `""`)+`")`)
	if err != nil {
		return false
	}
	defer func() { _ = rows.Close() }()
	present := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var dflt any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err == nil {
			present[strings.ToLower(name)] = true
		}
	}
	if err := rows.Err(); err != nil {
		return false
	}
	for _, column := range columns {
		if !present[strings.ToLower(column)] {
			return false
		}
	}
	return true
}
