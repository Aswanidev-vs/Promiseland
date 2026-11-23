-- Release 3 / 015: chat servers, server memberships and text channels.
-- All timestamps are UTC SQLite datetime('now') strings, as in 000_foundation.sql.

CREATE TABLE IF NOT EXISTS chat_servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    icon_url TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_chat_servers_owner ON chat_servers(owner_id);
CREATE INDEX IF NOT EXISTS idx_chat_servers_id ON chat_servers(id DESC);

CREATE TABLE IF NOT EXISTS chat_server_members (
    server_id INTEGER NOT NULL REFERENCES chat_servers(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'moderator', 'member')),
    joined_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_server_members_user
    ON chat_server_members(user_id, server_id);

CREATE TABLE IF NOT EXISTS chat_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL REFERENCES chat_servers(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL DEFAULT 'text' CHECK (type IN ('text')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (server_id, name)
);

CREATE INDEX IF NOT EXISTS idx_chat_channels_server
    ON chat_channels(server_id, id);
