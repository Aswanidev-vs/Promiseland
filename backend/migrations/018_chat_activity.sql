-- Release 3 / 018: chat activity state.
-- Scope columns are nullable pairs; the COALESCE-backed unique indexes make
-- (user, scope) genuinely unique because SQLite's composite PRIMARY KEY allows
-- multiple NULL values. Queries use "IS ?" so the same predicates work for both
-- integer and NULL scope values.

CREATE TABLE IF NOT EXISTS typing_state (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id INTEGER REFERENCES chat_channels(id) ON DELETE CASCADE,
    conversation_id INTEGER REFERENCES conversations(id) ON DELETE CASCADE,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, channel_id, conversation_id),
    CHECK ((channel_id IS NOT NULL AND conversation_id IS NULL)
        OR (channel_id IS NULL AND conversation_id IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_typing_scope
    ON typing_state(user_id, COALESCE(channel_id, -1), COALESCE(conversation_id, -1));
CREATE INDEX IF NOT EXISTS idx_typing_updated
    ON typing_state(updated_at);

-- Last message a user has read in each scope. Viewing a channel/conversation
-- advances this cursor, which is what drives the unread_count response fields.
CREATE TABLE IF NOT EXISTS chat_reads (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id INTEGER REFERENCES chat_channels(id) ON DELETE CASCADE,
    conversation_id INTEGER REFERENCES conversations(id) ON DELETE CASCADE,
    last_message_id INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, channel_id, conversation_id),
    CHECK ((channel_id IS NOT NULL AND conversation_id IS NULL)
        OR (channel_id IS NULL AND conversation_id IS NOT NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chat_reads_scope
    ON chat_reads(user_id, COALESCE(channel_id, -1), COALESCE(conversation_id, -1));

-- Lightweight heartbeat used by GET /chat/presence. The realtime hub pushes
-- authoritative online/offline events over the socket; this table only backs
-- the REST snapshot so a fresh page load can render presence immediately.
CREATE TABLE IF NOT EXISTS chat_presence (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'online' CHECK (status IN ('online', 'away', 'offline')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
