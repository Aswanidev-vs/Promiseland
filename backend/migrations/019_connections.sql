-- Release 4 / 019: friendship connection requests.
-- A pair of users may have at most one directed connection row; handlers also
-- reject the reverse direction so that one unordered pair has one request.

CREATE TABLE IF NOT EXISTS connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    requester_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'accepted', 'rejected', 'ignored')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (requester_id, addressee_id),
    CHECK (requester_id <> addressee_id)
);

CREATE INDEX IF NOT EXISTS idx_connections_addressee_status ON connections(addressee_id, status);
CREATE INDEX IF NOT EXISTS idx_connections_requester_status ON connections(requester_id, status);
CREATE INDEX IF NOT EXISTS idx_connections_status_id ON connections(status, id DESC);
