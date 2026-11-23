-- Release 2 / 011: event catalogue.
-- Times are stored as normalised RFC3339 UTC strings so lexicographic ordering
-- is chronological and cursor pagination by (starts_at, id) is stable.
-- Events are never hard-deleted by the API: cancelling keeps their history.

CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organizer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    event_type TEXT NOT NULL CHECK (event_type IN (
        'convention', 'meetup', 'screening', 'photoshoot',
        'club', 'online', 'competition', 'other')),
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    venue_text TEXT,
    city TEXT,
    country TEXT,
    is_online INTEGER NOT NULL DEFAULT 0 CHECK (is_online IN (0, 1)),
    -- 0 means "no capacity limit"; negative values are rejected at the API.
    capacity INTEGER NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    cover_media_id INTEGER REFERENCES media(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'cancelled', 'completed')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_starts_at ON events(starts_at, id);
CREATE INDEX IF NOT EXISTS idx_events_city ON events(city);
CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type, starts_at);
CREATE INDEX IF NOT EXISTS idx_events_organizer ON events(organizer_id, starts_at);
