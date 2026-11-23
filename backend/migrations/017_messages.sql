-- Release 3 / 017: persistent chat messages and per-user reaction codes.
-- A message belongs to exactly one scope: a channel or a conversation. The
-- CHECK keeps the two scope columns mutually exclusive and non-null. Messages
-- are soft-deleted (deleted_at) so thread integrity and moderation history are
-- preserved.

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id INTEGER REFERENCES chat_channels(id) ON DELETE CASCADE,
    conversation_id INTEGER REFERENCES conversations(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_message_id INTEGER REFERENCES messages(id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    edited_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    CHECK ((channel_id IS NOT NULL AND conversation_id IS NULL)
        OR (channel_id IS NULL AND conversation_id IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_messages_channel_created
    ON messages(channel_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_created
    ON messages(conversation_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_messages_parent
    ON messages(parent_message_id);

CREATE TABLE IF NOT EXISTS message_reactions (
    message_id INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reaction TEXT NOT NULL CHECK (reaction IN ('like', 'heart', 'star', 'laugh')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (message_id, user_id, reaction)
);

CREATE INDEX IF NOT EXISTS idx_message_reactions_message
    ON message_reactions(message_id, reaction);
