# johansenfoo

An agent-first personal site: a single Go binary that server-renders a public site from SQLite and
exposes a bcrypt-protected admin surface for editing it.

## Features

- Public landing page, `/me` JSON profile, `robots.txt` and `sitemap.xml`, served from an in-memory
  snapshot of the content stored in SQLite, plus a static `/health` liveness response.
- Content seeded from the original static site: profile, social links, projects, experience, skills
  and the `johansen` theme. The seed is close to, but not byte-identical with, the legacy export:
  some project wording follows the page copy rather than the legacy `me` file, and the ported markup
  differs slightly.
- A small in-memory content snapshot, reloaded after every admin write, so edits appear on the next
  page load without a restart.
- Admin surface at `/admin`: bcrypt password setup, login, logout and session cookies.
- CRUD for the profile, social links, projects, experience and skills, using plain POST forms (HTMX
  is vendored and loaded but not yet used).
- Password change and regenerable API key management at `/admin/security`.
- CSRF protection on stateful requests, a login rate limiter, and hourly pruning of expired
  sessions.
- A versioned REST API at `/api/v1`: unauthenticated reads for the profile, projects, experience,
  skills and theme, plus an admin surface authenticated by the API key or an admin session, with
  full CRUD and content export/import.
- No third-party origins on the critical path other than the Umami analytics snippet: icons are
  rendered as inline SVG, JetBrains Mono is self-hosted, and HTMX is vendored at
  `/static/htmx.min.js`.

## Admin

The site is edited through `/admin`, which does not exist until you create a password.

1. Start the service and open `http://localhost:8080/setup`.
2. Choose a password of at least 12 characters. It is stored as a bcrypt hash in the `settings` table.
3. From then on `/setup` redirects to `/login`.

Every write on an admin page reloads the content snapshot, so a change is visible on the public
site on the next page load. `/admin/security` changes the password and generates the API key that
the REST API and MCP server use (`Authorization: Bearer <key>`).

## API

The REST API lives under `/api/v1`. Public reads are unauthenticated and only return rows with
`visible = 1`; every `/api/v1/admin/*` route requires `Authorization: Bearer <api-key>` (the key
generated at `/admin/security`) or a valid admin session cookie.

```
Public
  GET  /api/v1/profile
  GET  /api/v1/projects
  GET  /api/v1/experience
  GET  /api/v1/skills
  GET  /api/v1/theme

Admin (Bearer or session)
  GET|PUT      /api/v1/admin/profile
  GET|POST     /api/v1/admin/social            PUT|DELETE /api/v1/admin/social/{id}
  GET|POST     /api/v1/admin/projects          GET|PUT|DELETE /api/v1/admin/projects/{id}
  GET|POST     /api/v1/admin/experience        GET|PUT|DELETE /api/v1/admin/experience/{id}
  GET|POST     /api/v1/admin/skills            GET|PUT|DELETE /api/v1/admin/skills/{id}
  PUT          /api/v1/admin/settings/posts    {"enabled": bool}
  PUT          /api/v1/admin/settings/theme    {"slug": string}
  GET          /api/v1/admin/export
  POST         /api/v1/admin/import
```

Writes require `Content-Type: application/json`. Creates return `201` with the created resource,
updates return `200` with the updated resource, deletes return `204`, and errors are
`{"error": "..."}` with an appropriate status. `GET /api/v1/admin/export` produces a whole-content
snapshot with the password hash and API key stripped; posting that snapshot back to
`/api/v1/admin/import` restores the content in one transaction.

## Build and run

Requires Go 1.25 and is CGO-free.

```sh
CGO_ENABLED=0 go build ./cmd/johansenfoo
./johansenfoo -data=./data
```

`-data` defaults to `./data`. An optional `config.json` in that directory overrides the defaults:

```json
{ "port": 8080, "db_path": "/path/to/johansenfoo.db", "base_url": "https://johansen.foo" }
```

## Architecture

```
cmd/johansenfoo/          entry point, -data flag, version, graceful shutdown
internal/config/          JSON config (port, db path, base URL)
internal/db/              SQLite (modernc, CGO-free), migrations and repositories
internal/db/migrations/   append-only schema, seed and theme migrations
internal/db/backup.go     whole-content export/import snapshot
internal/markdown/        goldmark wrapper (GFM, sanitised)
internal/theme/           token vocabulary, defaults, validation, CSS emission
internal/icons/           vendored SVGs exposed to templates as inline SVG
internal/web/             chi router, handlers, auth, CSRF, content snapshot
internal/web/api_*.go     /api/v1 public reads and authenticated admin REST handlers
internal/web/templates/   embedded html/template (public + admin)
internal/web/static/      embedded style.css, admin.css, fonts, htmx, favicons, avatar
```

## Tests

```sh
CGO_ENABLED=0 go test ./... -count=1
CGO_ENABLED=0 go vet ./...
gofmt -l .
```
