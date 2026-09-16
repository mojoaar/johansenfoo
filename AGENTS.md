# AGENTS.md

Guidance for agents working in this repository. The conventions mirror `icloud-mailflow`.

## Commands

```sh
CGO_ENABLED=0 go build ./...            # build every package (CGO-free)
CGO_ENABLED=0 go test ./... -count=1    # run the full test suite
CGO_ENABLED=0 go vet ./...              # static analysis
gofmt -l .                              # formatting check (must print nothing)
CGO_ENABLED=0 go run ./cmd/johansenfoo -data=/tmp/jf   # run against a scratch data dir
```

## Global constraints

- Module `github.com/mojoaar/johansenfoo`, Go 1.25.5, must build and test with `CGO_ENABLED=0`.
- No third-party origin on the runtime critical path other than the Umami snippet. HTMX is vendored
  at `/static/htmx.min.js`; no CDN scripts or fonts.
- Migrations are append-only: `0001`-`0005` stay byte-identical. Add the next free number only when
  required.
- All timestamps are RFC 3339 UTC via `strftime('%Y-%m-%dT%H:%M:%SZ','now')`, never `datetime('now')`.
- All SQL is parameterized.
- `<html>` carries `data-theme="<slug>"` and `data-mode="light|dark"` as separate axes.
- No comments in Go code unless essential (`//go:embed` excepted).

## Architecture

```
cmd/johansenfoo/main.go     entry point, -data/./data flag, version, graceful shutdown
internal/config/            JSON config (port, db path, base URL)
internal/db/                SQLite (modernc, CGO-free) + migrations + repositories
internal/db/migrate.go      append-only migration runner
internal/db/repo_profile.go profile and social-link repository
internal/db/repo_content.go project, experience and skill repository
internal/db/repo_settings.go key/value settings repository
internal/db/repo_theme.go   theme repository
internal/db/repo_session.go session repository: Create, Get, Delete, DeleteExpired
internal/markdown/          goldmark wrapper (GFM, sanitised)
internal/theme/             token vocabulary, defaults, validation, CSS emission
internal/icons/             vendored SVGs, exposed to templates as inline SVG
internal/web/               chi router, middleware, handlers and the content snapshot
internal/web/content.go     ContentStore: cached SiteContent snapshot with Reload
internal/web/landing.go     LoadContent and the public landing handler
internal/web/me.go          /me JSON profile
internal/web/seo.go         robots.txt, sitemap.xml and page metadata
internal/web/health.go      static liveness response for /health
internal/web/render.go      page data, embedded templates, renderPage/renderAdmin
internal/web/static.go      embedded static assets; directory listings are rejected
internal/web/auth.go        bcrypt password, sessions, auth middleware, session pruning
internal/web/csrf.go        CSRF middleware and POST method override
internal/web/admin.go       admin shell and dashboard
internal/web/admin_profile.go   profile and social-link handlers
internal/web/admin_projects.go  project CRUD handlers
internal/web/admin_experience.go experience CRUD handlers
internal/web/admin_skills.go    skill CRUD handlers
internal/web/admin_security.go  password change and API-key regeneration
internal/web/templates/     embedded html/template (public + admin)
internal/web/static/        embedded style.css, admin.css, fonts, htmx, favicons, avatar
```

Public routes: `GET /`, `GET /me`, `GET /robots.txt`, `GET /sitemap.xml`, `GET /health`,
`GET|POST /setup`, `GET|POST /login`, `POST /logout`.

Admin routes (all behind authMiddleware, HTMX-driven forms):

```
  GET  /admin                  dashboard
  GET  /admin/profile          POST /admin/profile
  GET  /admin/social           POST /admin/social
                               POST /admin/social/{id}
                               POST /admin/social/{id}/delete
  GET  /admin/projects         POST /admin/projects
  GET  /admin/projects/new
  GET  /admin/projects/{id}    POST /admin/projects/{id}
                               POST /admin/projects/{id}/delete
  GET  /admin/experience       POST /admin/experience
  GET  /admin/experience/new
  GET  /admin/experience/{id}  POST /admin/experience/{id}
                               POST /admin/experience/{id}/delete
  GET  /admin/skills           POST /admin/skills
                               POST /admin/skills/{id}
                               POST /admin/skills/{id}/delete
  GET  /admin/security
                               POST /admin/security/password
                               POST /admin/security/apikey
Auth routes: GET|POST /setup, GET|POST /login, POST /logout
```

## Behaviour to preserve

- Every successful admin write calls `d.Content.Reload()` so the next public request reflects it.
- Every non-HTMX admin write form carries a server-rendered hidden `csrf_token`. Requests with
  `HX-Request: true`, GET/HEAD, and POSTs to `/login`, `/setup` and `/mcp` are CSRF-exempt.
- Repo list methods return every row plus a `Visible` field; visibility is filtered in
  `render.go` / `me.go` / `seo.go`, never with `WHERE visible = 1`.
- Documentation is part of the definition of done: a new feature updates the README feature list, a
  new package updates the architecture trees here and in the README, and changed routes update both.
