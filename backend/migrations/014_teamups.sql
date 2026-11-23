-- Release 2 / 014: collaboration / Team Up requests.
--
-- The events table is created by migration 011 (same release, applied first).
-- SQLite accepts a forward reference at CREATE TABLE time, so this migration
-- still applies if the events migration is not present yet; the team-ups API
-- treats event_id as optional and validates it against the events table when
-- one is supplied.

CREATE TABLE IF NOT EXISTS team_ups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_id INTEGER REFERENCES events(id) ON DELETE SET NULL,
    -- JSON array of category tokens (creator categories and event tags).
    categories TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'filled', 'closed')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_team_ups_owner_id ON team_ups(owner_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_team_ups_status_id ON team_ups(status, id DESC);
CREATE INDEX IF NOT EXISTS idx_team_ups_event_id ON team_ups(event_id);

-- One row per (team_up, applicant) is the dedup rule. A separate AUTOINCREMENT
-- id is exposed so clients can address a specific application when reviewing.
CREATE TABLE IF NOT EXISTS team_up_applications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    team_up_id INTEGER NOT NULL REFERENCES team_ups(id) ON DELETE CASCADE,
    applicant_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (team_up_id, applicant_id)
);

CREATE INDEX IF NOT EXISTS idx_team_up_applications_team
    ON team_up_applications(team_up_id, id);
CREATE INDEX IF NOT EXISTS idx_team_up_applications_applicant
    ON team_up_applications(applicant_id, id DESC);
