CREATE TABLE IF NOT EXISTS page_view (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL,
    referrer   TEXT,
    user_agent TEXT,
    ip_hash    TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_page_view_created_at ON page_view (created_at);
