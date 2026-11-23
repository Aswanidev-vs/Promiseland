-- Release 2 / 012: event RSVPs, organiser updates and discussion comments.
-- Activity is cascaded away only if an event row is physically deleted; the
-- API cancels events instead, so history remains queryable.

CREATE TABLE IF NOT EXISTS event_rsvps (
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('going', 'maybe')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_event_rsvps_user ON event_rsvps(user_id, event_id);
CREATE INDEX IF NOT EXISTS idx_event_rsvps_status ON event_rsvps(event_id, status);

CREATE TABLE IF NOT EXISTS event_updates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_event_updates_event ON event_updates(event_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS event_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_comment_id INTEGER REFERENCES event_comments(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    deleted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_event_comments_event ON event_comments(event_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_event_comments_parent ON event_comments(parent_comment_id);
