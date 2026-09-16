# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Admin surface at `/admin` with bcrypt password setup, login, logout and session cookies.
- HTMX-driven CRUD for the profile, social links, projects, experience and skills.
- Password change and regenerable API key management at `/admin/security`.
- CSRF protection on stateful requests and a login rate limiter.
- Session table with hourly pruning of expired sessions.

### Changed
- Public pages now reload site content from SQLite after every write instead of serving a
  startup snapshot, so admin edits appear on the next page load.
- Static assets reject directory listings.

## [0.1.0] - 2026-09-16

### Added
- Public site rendered from SQLite: landing page, `/me`, `robots.txt`, `sitemap.xml` and `/health`.
- Append-only schema, seed and theme migrations, with self-hosted fonts and inline SVG icons.
