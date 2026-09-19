CREATE TABLE IF NOT EXISTS page_seo (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    route         TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    og_image_url  TEXT NOT NULL DEFAULT '',
    canonical_url TEXT NOT NULL DEFAULT '',
    noindex       INTEGER NOT NULL DEFAULT 0
);
