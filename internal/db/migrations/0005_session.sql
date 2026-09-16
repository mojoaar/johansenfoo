CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_expires_at ON session (expires_at);
