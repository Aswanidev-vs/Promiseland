-- Release 3 / 016: direct and group conversations.
-- A direct conversation is unique per unordered pair of users. pair_key is
-- stored as "<lower id>:<higher id>" and is only populated for direct chats;
-- the partial unique index is the database-level backstop for the lookup that
-- the chat handlers perform before inserting.

CREATE TABLE IF NOT EXISTS conversations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL CHECK (type IN ('direct', 'group')),
    name TEXT NOT NULL DEFAULT '',
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    pair_key TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_pair_key
    ON conversations(pair_key) WHERE pair_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_conversations_type_id
    ON conversations(type, id DESC);

CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_members_user
    ON conversation_members(user_id, conversation_id);
