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
- Migrations are append-only: `0001`-`0006` stay byte-identical. Add the next free number (`0007`)
  only when required.
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
internal/db/repo_post.go    post, tag and post_tag repository
internal/db/backup.go       whole-content export/import snapshot (includes posts/tags)
internal/markdown/          goldmark + GFM + Chroma class-based highlighting
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
internal/web/api.go         JSON helpers (writeJSON, writeAPIError, decodeJSON, apiID)
internal/web/api_public.go  public /api/v1 read handlers
internal/web/api_auth.go    API session/Bearer authentication middleware
internal/web/api_admin_profile.go  admin profile REST handlers
internal/web/api_admin_social.go   admin social-link REST handlers
internal/web/api_admin_content.go  generic admin REST CRUD for projects/experience/skills
internal/web/api_admin_settings.go settings REST handlers
internal/web/api_admin_backup.go   export/import REST handlers
internal/web/posts.go       public post index, single post, tag archive, date helpers
internal/web/feed.go        RSS 2.0 feed
internal/web/admin_posts.go admin post CRUD and publish controls
internal/web/api_posts.go   public posts REST handlers
internal/web/api_admin_posts.go admin posts REST handlers
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
`GET /posts`, `GET /posts/{slug}`, `GET /tags/{slug}`, `GET /feed.xml`,
`GET|POST /setup`, `GET|POST /login`, `POST /logout`.

Admin routes (all behind authMiddleware; the templates post plain forms, and the PUT/DELETE
variants are reachable via the `_method` override):

```
  GET  /admin                  dashboard
  GET  /admin/profile          POST|PUT /admin/profile
  GET  /admin/social           POST /admin/social
                               POST|PUT /admin/social/{id}
                               POST|DELETE /admin/social/{id}/delete
  GET  /admin/projects         POST /admin/projects
  GET  /admin/projects/new
  GET  /admin/projects/{id}    POST|PUT /admin/projects/{id}
                               POST|DELETE /admin/projects/{id}/delete
  GET  /admin/experience       POST /admin/experience
  GET  /admin/experience/new
  GET  /admin/experience/{id}  POST|PUT /admin/experience/{id}
                               POST|DELETE /admin/experience/{id}/delete
  GET  /admin/skills           POST /admin/skills
                               POST|PUT /admin/skills/{id}
                               POST|DELETE /admin/skills/{id}/delete
  GET  /admin/security
                               POST|PUT /admin/security/password
                               POST|PUT /admin/security/apikey
  GET  /admin/posts            POST /admin/posts
  GET  /admin/posts/new
  GET  /admin/posts/{id}       POST|PUT /admin/posts/{id}
                               POST|DELETE /admin/posts/{id}/delete
                               POST|PUT /admin/posts/{id}/publish
                               POST|PUT /admin/posts/{id}/unpublish
Auth routes: GET|POST /setup, GET|POST /login, POST /logout
```

API routes:

```
  Public: GET /api/v1/profile|projects|experience|skills|theme
  Public: GET /api/v1/posts (?page,&tag=) and /api/v1/posts/{slug}
  Admin (all behind apiAuthMiddleware; Bearer API key or admin session cookie):
    GET|PUT      /api/v1/admin/profile
    GET|POST     /api/v1/admin/social            PUT|DELETE /api/v1/admin/social/{id}
    GET|POST     /api/v1/admin/projects|experience|skills
    GET|PUT|DELETE /api/v1/admin/projects|experience|skills/{id}
    GET|POST     /api/v1/admin/posts             GET|PUT|DELETE /api/v1/admin/posts/{id}
    POST         /api/v1/admin/posts/{id}/publish   /unpublish
    PUT          /api/v1/admin/settings/posts    PUT /api/v1/admin/settings/theme
    GET          /api/v1/admin/export            POST /api/v1/admin/import
```

## Behaviour to preserve

- Every successful admin write calls `d.Content.Reload()` so the next public request reflects it.
- Every non-HTMX admin write form carries a server-rendered hidden `csrf_token`. Requests with
  `HX-Request: true`, GET/HEAD, and POSTs to `/login`, `/setup` and `/mcp` are CSRF-exempt.
- Repo list methods return every row plus a `Visible` field; visibility is filtered in
  `render.go` / `me.go` / `seo.go` / `api_public.go`, never with `WHERE visible = 1`.
- Public `/api/v1` reads filter `Visible`; admin `/api/v1/admin/*` reads return every row.
- Public post surfaces (`/posts`, `/posts/{slug}`, `/tags/{slug}`, `/feed.xml`, `/api/v1/posts*`,
  `sitemap.xml`) only ever read `status = 'published'` posts; a draft slug returns 404.
- `posts_enabled=false` 404s every public post surface and hides the nav link, but never deletes
  data. `sitemap_enabled=false` 404s `/sitemap.xml`.
- Every successful admin API write calls `d.Content.Reload()`, exactly like the HTML admin handlers.
- `/api/` paths are CSRF-exempt; API write safety rests on Bearer/JSON rather than a form token.
- Documentation is part of the definition of done: a new feature updates the README feature list, a
  new package updates the architecture trees here and in the README, and changed routes update both.
