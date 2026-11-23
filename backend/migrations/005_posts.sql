-- Release 1: posts, comments, votes, bookmarks and polls.
-- Content is soft-deleted so that discussion threads stay intact.

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    community_id INTEGER REFERENCES communities(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('text', 'link', 'image', 'question', 'poll', 'announcement')),
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    link_url TEXT,
    spoiler INTEGER NOT NULL DEFAULT 0 CHECK (spoiler IN (0, 1)),
    spoiler_note TEXT,
    -- poll_question is optional; polls fall back to the post title.
    poll_question TEXT,
    season INTEGER,
    episode INTEGER,
    -- anime_ref_id is a legacy single reference; post_anime is authoritative.
    -- No FK: anime titles are owned by a different (Release 1) module/migration.
    anime_ref_id INTEGER,
    vote_score INTEGER NOT NULL DEFAULT 0,
    is_locked INTEGER NOT NULL DEFAULT 0 CHECK (is_locked IN (0, 1)),
    deleted_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT,
    edited_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_posts_community_id ON posts(community_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_posts_author_id ON posts(author_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_posts_live_id ON posts(deleted_at, id DESC);
CREATE INDEX IF NOT EXISTS idx_posts_type_id ON posts(type, id DESC);

CREATE TABLE IF NOT EXISTS post_media (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, media_id)
);

CREATE INDEX IF NOT EXISTS idx_post_media_media ON post_media(media_id);

CREATE TABLE IF NOT EXISTS post_anime (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    anime_title_id INTEGER NOT NULL,
    PRIMARY KEY (post_id, anime_title_id)
);

CREATE TABLE IF NOT EXISTS post_tags (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag TEXT NOT NULL,
    PRIMARY KEY (post_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_post_tags_tag ON post_tags(tag);

CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_comment_id INTEGER REFERENCES comments(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    edited_at TEXT,
    deleted_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_comment_id);

CREATE TABLE IF NOT EXISTS votes (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    value INTEGER NOT NULL CHECK (value IN (-1, 1)),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_post ON votes(post_id);

CREATE TABLE IF NOT EXISTS bookmarks (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_post ON bookmarks(post_id);

CREATE TABLE IF NOT EXISTS poll_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_poll_options_post ON poll_options(post_id, sort_order, id);

CREATE TABLE IF NOT EXISTS poll_votes (
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    option_id INTEGER NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_poll_votes_option ON poll_votes(option_id);
