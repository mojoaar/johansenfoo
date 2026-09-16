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
the planned REST API and MCP server will use (`Authorization: Bearer <key>`).

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
internal/markdown/        goldmark wrapper (GFM, sanitised)
internal/theme/           token vocabulary, defaults, validation, CSS emission
internal/icons/           vendored SVGs exposed to templates as inline SVG
internal/web/             chi router, handlers, auth, CSRF, content snapshot
internal/web/templates/   embedded html/template (public + admin)
internal/web/static/      embedded style.css, admin.css, fonts, htmx, favicons, avatar
```

## Tests

```sh
CGO_ENABLED=0 go test ./... -count=1
CGO_ENABLED=0 go vet ./...
gofmt -l .
```
