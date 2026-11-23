-- Release 5 / 021: optional, adults-only dating scaffold (PRD §7.14).
--
-- SAFETY: this schema is inert unless FEATURE_ADULT_DATING is explicitly
-- enabled. It deliberately contains NO location, phone, email, contact, or
-- birthday fields: a self-entered birthday is not sufficient age assurance for
-- a high-risk dating feature. A real age-assurance provider must be integrated
-- before this area can be used for real. Eligibility is stored in
-- adult_verifications and is denied by default.

CREATE TABLE IF NOT EXISTS adult_profiles (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    intent TEXT NOT NULL CHECK (intent IN ('friendship', 'dating', 'both')),
    bio TEXT NOT NULL DEFAULT '',
    boundaries TEXT NOT NULL DEFAULT '',
    -- Opt-in is the only supported visibility. There is no public discovery
    -- listing for adult profiles; profiles are reachable only by their owner
    -- and by users who have mutually matched.
    visibility TEXT NOT NULL DEFAULT 'opt_in' CHECK (visibility = 'opt_in'),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

CREATE TABLE IF NOT EXISTS adult_verifications (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'unverified' CHECK (status IN ('unverified', 'verified')),
    method TEXT NOT NULL DEFAULT 'none',
    verified_at TEXT,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_adult_verifications_status
    ON adult_verifications(status, expires_at);

CREATE TABLE IF NOT EXISTS adult_likes (
    actor_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('like', 'pass')),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (actor_id, target_id),
    CHECK (actor_id <> target_id)
);

-- actor_id lookups are served by the primary key; this index covers "who
-- liked/matched this user" checks and mutual-like lookups.
CREATE INDEX IF NOT EXISTS idx_adult_likes_target_status
    ON adult_likes(target_id, status, actor_id);
CREATE INDEX IF NOT EXISTS idx_adult_likes_created
    ON adult_likes(created_at);

CREATE TABLE IF NOT EXISTS adult_matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_a INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    unmatched_at TEXT,
    UNIQUE (user_a, user_b)
);

CREATE INDEX IF NOT EXISTS idx_adult_matches_user_a ON adult_matches(user_a, unmatched_at, created_at);
CREATE INDEX IF NOT EXISTS idx_adult_matches_user_b ON adult_matches(user_b, unmatched_at, created_at);
CREATE INDEX IF NOT EXISTS idx_adult_matches_created ON adult_matches(created_at);
