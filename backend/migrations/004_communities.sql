-- Release 1: communities, memberships and rules.
-- All timestamps are UTC SQLite datetime('now') strings, as in 000_foundation.sql.

CREATE TABLE IF NOT EXISTS communities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    icon_url TEXT,
    banner_url TEXT,
    visibility TEXT NOT NULL DEFAULT 'public' CHECK (visibility IN ('public', 'private')),
    posting_enabled INTEGER NOT NULL DEFAULT 1 CHECK (posting_enabled IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_communities_owner ON communities(owner_id);
CREATE INDEX IF NOT EXISTS idx_communities_visibility_id ON communities(visibility, id DESC);

CREATE TABLE IF NOT EXISTS community_memberships (
    community_id INTEGER NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'moderator', 'member')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'pending')),
    joined_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (community_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_status ON community_memberships(user_id, status);
CREATE INDEX IF NOT EXISTS idx_memberships_community_status ON community_memberships(community_id, status);

CREATE TABLE IF NOT EXISTS community_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    community_id INTEGER NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_community_rules_community ON community_rules(community_id, sort_order, id);
