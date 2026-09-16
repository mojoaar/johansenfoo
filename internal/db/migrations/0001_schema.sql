CREATE TABLE IF NOT EXISTS profile (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    name          TEXT NOT NULL,
    handle        TEXT NOT NULL,
    location      TEXT NOT NULL,
    dob           TEXT NOT NULL,
    tagline       TEXT NOT NULL,
    hero_bio      TEXT NOT NULL,
    about_para_1  TEXT NOT NULL,
    about_para_2  TEXT NOT NULL,
    avatar        TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS social_link (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    platform TEXT NOT NULL,
    url      TEXT NOT NULL,
    label    TEXT NOT NULL,
    sort     INTEGER NOT NULL DEFAULT 0,
    visible  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS project (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    url         TEXT,
    description TEXT NOT NULL,
    icon        TEXT NOT NULL,
    is_link     INTEGER NOT NULL DEFAULT 1,
    url_label   TEXT NOT NULL DEFAULT '',
    sort        INTEGER NOT NULL DEFAULT 0,
    visible     INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS experience (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    years   TEXT NOT NULL,
    role    TEXT NOT NULL,
    company TEXT NOT NULL,
    icon    TEXT NOT NULL,
    sort    INTEGER NOT NULL DEFAULT 0,
    visible INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS skill (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT NOT NULL,
    sort    INTEGER NOT NULL DEFAULT 0,
    visible INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS theme (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    tokens_base  TEXT NOT NULL DEFAULT '{}',
    tokens_light TEXT NOT NULL DEFAULT '{}',
    tokens_dark  TEXT NOT NULL DEFAULT '{}',
    sort         INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
