-- Release 1: user reports and the platform-moderator grant list.
-- Platform moderation is represented by a dedicated grant table rather than a
-- column on users: it avoids colliding with other R1 migrations that extend
-- the users table, and it keeps moderation state in the moderation domain.
-- No regular account is a moderator by default.

CREATE TABLE IF NOT EXISTS platform_moderators (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Other R1 modules detect this flag directly; it is off for every account
-- unless the dev-only bootstrap or a moderator grant sets it.
ALTER TABLE users ADD COLUMN is_moderator INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reporter_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_type TEXT NOT NULL,
    target_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    details TEXT,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved', 'dismissed')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    resolved_at TEXT,
    resolved_by INTEGER REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_reports_status_created
    ON reports(status, created_at);

-- A reporter may only have one open report per target at a time.
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_open_unique
    ON reports(reporter_id, target_type, target_id)
    WHERE status = 'open';
