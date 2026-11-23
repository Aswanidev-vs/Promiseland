-- Release 2 / 013: creator categories and curated featured posts.
-- Applied after the Release 1 post schema (005_posts.sql).

ALTER TABLE posts ADD COLUMN creator_category TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_posts_creator_category ON posts(creator_category, id DESC);
CREATE INDEX IF NOT EXISTS idx_posts_author_category ON posts(author_id, creator_category, id DESC);

-- A user may feature at most six of their own posts. Ownership is verified by
-- the creators handler; the FK keeps the link table clean when posts/users go
-- away.
CREATE TABLE IF NOT EXISTS featured_posts (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_featured_posts_user_sort
    ON featured_posts(user_id, sort_order, post_id);
