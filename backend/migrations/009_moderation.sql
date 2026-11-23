-- Release 1: moderation audit trail and account suspensions.

CREATE TABLE IF NOT EXISTS moderation_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    moderator_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    scope TEXT NOT NULL CHECK (scope IN ('community', 'platform')),
    target_type TEXT NOT NULL,
    target_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    reason TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_moderation_actions_created
    ON moderation_actions(created_at DESC);

CREATE TABLE IF NOT EXISTS suspensions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_suspensions_user_active
    ON suspensions(user_id, is_active, expires_at);
