-- Release 1: in-app notifications and per-category preferences.
-- CREATE TABLE IF NOT EXISTS keeps this safe on databases where an earlier
-- migration already created a compatible notifications table.

CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    actor_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    target_type TEXT NOT NULL,
    target_id INTEGER NOT NULL,
    data TEXT NOT NULL DEFAULT '{}',
    read_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_created
    ON notifications(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS notification_preferences (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, category)
);

-- Seed default-enabled preferences. INSERT OR IGNORE preserves any choice a
-- user has already made and is safe to rerun.
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'follow', 1 FROM users;
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'reply', 1 FROM users;
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'mention', 1 FROM users;
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'reaction', 1 FROM users;
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'community', 1 FROM users;
INSERT OR IGNORE INTO notification_preferences (user_id, category, enabled)
SELECT id, 'event', 1 FROM users;
