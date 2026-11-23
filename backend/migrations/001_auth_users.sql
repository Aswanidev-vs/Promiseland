-- Release 1 / 001: social identity for accounts.
-- Applied after 000_foundation.sql. The migration runner records each file in
-- schema_migrations, so this file is only applied once per database.

-- SQLite cannot add a UNIQUE column directly, so username is added nullable and
-- its uniqueness is enforced by a unique index created after the backfill.
ALTER TABLE users ADD COLUMN username TEXT;
ALTER TABLE users ADD COLUMN discoverable INTEGER NOT NULL DEFAULT 1;

ALTER TABLE user_profiles ADD COLUMN favorite_characters TEXT;
ALTER TABLE user_profiles ADD COLUMN external_links TEXT;

-- Backfill accounts created with the 000-only schema. Handles are derived from
-- the display name and suffixed with the immutable user id so they are always
-- unique. Kept short (<= 15 characters + id) to respect the 3..20 rule.
UPDATE users
SET username = COALESCE(
        NULLIF(substr(lower(replace(replace(replace(replace(replace(display_name, ' ', '_'), '.', ''), '@', ''), '-', '_'), '''', '')), 1, 15), ''),
        'user_'
    ) || id
WHERE username IS NULL OR trim(username) = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_discoverable ON users(discoverable);

CREATE TABLE IF NOT EXISTS interests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    UNIQUE (name, category)
);

INSERT OR IGNORE INTO interests (name, category) VALUES
    ('cosplay', 'creative'),
    ('crossplay', 'creative'),
    ('fan art', 'creative'),
    ('photography', 'creative'),
    ('manga', 'reading'),
    ('conventions', 'community'),
    ('gaming', 'hobbies'),
    ('edits', 'creative'),
    ('collections', 'hobbies'),
    ('voice acting', 'creative'),
    ('figure collecting', 'hobbies'),
    ('webtoons', 'reading'),
    ('J-pop', 'music'),
    ('cosplay props', 'creative');

CREATE TABLE IF NOT EXISTS user_interests (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    interest_id INTEGER NOT NULL REFERENCES interests(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, interest_id)
);

CREATE INDEX IF NOT EXISTS idx_user_interests_interest ON user_interests(interest_id);

-- anime_titles is created by 002_anime.sql (applied next). This table therefore
-- keeps anime_title_id as a plain integer reference; the API validates the id
-- against anime_titles on write.
CREATE TABLE IF NOT EXISTS anime_interests (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    anime_title_id INTEGER NOT NULL,
    interest_type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, anime_title_id, interest_type)
);

CREATE INDEX IF NOT EXISTS idx_anime_interests_title ON anime_interests(anime_title_id);

CREATE TABLE IF NOT EXISTS blocks (
    blocker_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (blocker_id, blocked_id),
    CHECK (blocker_id <> blocked_id)
);

CREATE INDEX IF NOT EXISTS idx_blocks_blocked ON blocks(blocked_id);

-- notification_preferences is intentionally not defined here: the notifications
-- module owns that table (see 007_notifications.sql).
