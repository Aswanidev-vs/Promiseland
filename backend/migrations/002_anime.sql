-- Release 1 / 002: small, seeded anime catalogue.
-- Only public bibliographic metadata (title, alternate title, format) is
-- stored. No artwork, episode data or unverified external identifiers.

CREATE TABLE IF NOT EXISTS anime_titles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    external_id TEXT,
    title TEXT NOT NULL,
    alternate_title TEXT,
    format TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (title, format)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_anime_external_id
    ON anime_titles(external_id) WHERE external_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_anime_title ON anime_titles(title);
CREATE INDEX IF NOT EXISTS idx_anime_alternate_title ON anime_titles(alternate_title);

CREATE TABLE IF NOT EXISTS genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS anime_genres (
    anime_id INTEGER NOT NULL REFERENCES anime_titles(id) ON DELETE CASCADE,
    genre_id INTEGER NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
    PRIMARY KEY (anime_id, genre_id)
);

INSERT OR IGNORE INTO genres (name) VALUES
    ('Action'),
    ('Adventure'),
    ('Comedy'),
    ('Drama'),
    ('Fantasy'),
    ('Horror'),
    ('Mecha'),
    ('Mystery'),
    ('Romance'),
    ('Sci-Fi'),
    ('Slice of Life'),
    ('Sports'),
    ('Supernatural'),
    ('Thriller');

INSERT OR IGNORE INTO anime_titles (title, alternate_title, format) VALUES
    ('Attack on Titan', 'Shingeki no Kyojin', 'TV'),
    ('Cowboy Bebop', 'Cowboy Bebop', 'TV'),
    ('Demon Slayer: Kimetsu no Yaiba', 'Kimetsu no Yaiba', 'TV'),
    ('Death Note', 'Death Note', 'TV'),
    ('Dragon Ball', 'Dragon Ball', 'TV'),
    ('Fullmetal Alchemist: Brotherhood', 'Hagane no Renkinjutsushi', 'TV'),
    ('Haikyu!!', 'Haikyuu', 'TV'),
    ('Jujutsu Kaisen', 'Jujutsu Kaisen', 'TV'),
    ('Mob Psycho 100', 'Mob Psycho 100', 'TV'),
    ('My Hero Academia', 'Boku no Hero Academia', 'TV'),
    ('Naruto', 'NARUTO', 'TV'),
    ('Naruto: Shippuden', 'NARUTO Shippuuden', 'TV'),
    ('Neon Genesis Evangelion', 'Shin Seiki Evangelion', 'TV'),
    ('One Piece', 'ONE PIECE', 'TV'),
    ('Sailor Moon', 'Bishoujo Senshi Sailor Moon', 'TV'),
    ('Spy x Family', 'SPYxFAMILY', 'TV'),
    ('Spirited Away', 'Sen to Chihiro no Kamikakushi', 'Film'),
    ('Your Name', 'Kimi no Na wa', 'Film');

-- A few factual genre links so genre-filtered UI work has data.
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Attack on Titan' AND g.name = 'Action';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Attack on Titan' AND g.name = 'Drama';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Cowboy Bebop' AND g.name = 'Sci-Fi';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Death Note' AND g.name = 'Mystery';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Fullmetal Alchemist: Brotherhood' AND g.name = 'Fantasy';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Haikyu!!' AND g.name = 'Sports';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Jujutsu Kaisen' AND g.name = 'Supernatural';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Naruto' AND g.name = 'Action';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Sailor Moon' AND g.name = 'Romance';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Sailor Moon' AND g.name = 'Fantasy';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Spy x Family' AND g.name = 'Comedy';
INSERT OR IGNORE INTO anime_genres (anime_id, genre_id)
    SELECT a.id, g.id FROM anime_titles a, genres g
    WHERE a.title = 'Spirited Away' AND g.name = 'Adventure';
