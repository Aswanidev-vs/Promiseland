-- Release 4 / 020: reversible "hide this suggestion" state for discovery.
-- No location data is stored or derived anywhere in discovery.

CREATE TABLE IF NOT EXISTS discovery_suppressions (
    viewer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    candidate_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (viewer_id, candidate_id),
    CHECK (viewer_id <> candidate_id)
);

CREATE INDEX IF NOT EXISTS idx_discovery_suppressions_candidate ON discovery_suppressions(candidate_id);
