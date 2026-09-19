# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Public read API at `/api/v1` for the profile, projects, experience, skills and theme.
- Authenticated admin REST API at `/api/v1/admin` with CRUD for the profile, social links,
  projects, experience and skills, plus posts/theme settings.
- Whole-content export and import at `/api/v1/admin/export` and `/api/v1/admin/import`, with the
  password hash and API key stripped from exports.
- Admin surface at `/admin` with bcrypt password setup, login, logout and session cookies.
- CRUD for the profile, social links, projects, experience and skills.
- Password change and regenerable API key management at `/admin/security`.
- CSRF protection on stateful requests and a login rate limiter.
- Session table with hourly pruning of expired sessions.

### Changed
- Public pages now reload site content from SQLite after every write instead of serving a
  startup snapshot, so admin edits appear on the next page load.
- Static assets reject directory listings.

## [0.1.0] - 2026-09-16

No git tag exists for this version yet; the date records when this work landed.

### Added
- Public site rendered from SQLite: landing page, `/me`, `robots.txt`, `sitemap.xml` and `/health`.
- Append-only schema, seed and theme migrations, with self-hosted fonts and inline SVG icons.
