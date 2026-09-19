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
  skills, theme and posts, plus an admin surface authenticated by the API key or an admin session,
  with full CRUD and content export/import.
- A Posts system: markdown posts with server-side syntax highlighting, tags, drafts, hero images,
  pagination, an RSS feed at `/feed.xml`, and a `posts_enabled` kill switch.
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
  GET  /api/v1/posts                            ?page=&tag=
  GET  /api/v1/posts/{slug}

Admin (Bearer or session)
  GET|PUT      /api/v1/admin/profile
  GET|POST     /api/v1/admin/social            PUT|DELETE /api/v1/admin/social/{id}
  GET|POST     /api/v1/admin/projects          GET|PUT|DELETE /api/v1/admin/projects/{id}
  GET|POST     /api/v1/admin/experience        GET|PUT|DELETE /api/v1/admin/experience/{id}
  GET|POST     /api/v1/admin/skills            GET|PUT|DELETE /api/v1/admin/skills/{id}
  GET|POST     /api/v1/admin/posts             GET|PUT|DELETE /api/v1/admin/posts/{id}
  POST         /api/v1/admin/posts/{id}/publish
  POST         /api/v1/admin/posts/{id}/unpublish
  PUT          /api/v1/admin/settings/posts    {"enabled": bool}
  PUT          /api/v1/admin/settings/theme    {"slug": string}
  GET|PUT      /api/v1/admin/settings/seo
  GET          /api/v1/admin/export
  POST         /api/v1/admin/import
```

Writes require `Content-Type: application/json`. Creates return `201` with the created resource,
updates return `200` with the updated resource, deletes return `204`, and errors are
`{"error": "..."}` with an appropriate status. `PUT` is a full replace: send every field, not just
the changed one. `GET /api/v1/admin/export` produces a whole-content snapshot with the password
hash and API key stripped; posting that snapshot back to `/api/v1/admin/import` restores the
content in one transaction. An import must carry every content section and a valid `active_theme`,
or it is rejected without changing anything.

Post payloads accept `tags` as an array of strings (`["go","testing"]`) or of `{name, slug}`
objects. Creating or renaming a post to a slug that is already in use returns `409`; an empty slug
is rejected. `PUT` that omits `status` keeps the post's current status. Snapshots are versioned:
a snapshot exported by an older version is rejected, so re-export after upgrading.
Public post timestamps are emitted in the site timezone with an offset
(`2026-09-16T14:30:00+02:00`). `PUT /api/v1/admin/settings/seo` is a full replace and requires the
`pages` array (the `GET` response always includes it); a snapshot exported by an older version is
rejected, so re-export after upgrading.

## Posts

Posts are authored at `/admin/posts`, where each post has a title, slug, summary, markdown body,
tags, an optional hero image, SEO fields, and a draft/published status. Only published posts are
visible on the public surface.

```
  GET /posts                 paginated index, ?page=, ?tag=
  GET /posts/{slug}          single published post
  GET /tags/{slug}           tag archive
  GET /feed.xml              RSS 2.0
```

Code blocks are highlighted server-side with Chroma; `/static/code.css` maps Chroma's classes to
the theme tokens, so highlighting follows the light/dark mode with no JavaScript. The
`posts_enabled` setting (editable at `PUT /api/v1/admin/settings/posts`) is a kill switch: when it
is `false`, every public post surface returns `404`, the nav link is hidden and the sitemap omits
posts, but the posts stay in SQLite and return when it is re-enabled. `sitemap_enabled=false` makes
`/sitemap.xml` return `404`.

## SEO

Metadata is resolved with a clear precedence: a per-page `page_seo` row or a per-post override
beats the global SEO defaults, which beat the hardcoded fallback. All of it is editable at
`/admin/seo` or through `GET|PUT /api/v1/admin/settings/seo`.

- **Global defaults** — site title, title template (`%s | johansen.foo`), default meta description,
  default OG image, OG type, Twitter card and site, canonical base URL, `robots.txt`, a site-wide
  `noindex` toggle, and a sitemap toggle.
- **Per-page overrides** live in `page_seo`, keyed by route (`/`, `/posts`, `/tags/{slug}`).
- **Per-post overrides** — a post's `seo_title`, `seo_description`, `og_image_url`, `canonical_url`
  and `noindex`; its description defaults to the summary and its OG image to the hero image.
- Every published post also emits `BlogPosting` structured data; the home page emits `Person`.

## MCP

The same content is exposed over MCP streamable HTTP at `/mcp`, authenticated with the API key
(`Authorization: Bearer <api-key>`) and limited to 100 requests per minute per IP. Tool names are
snake_case and errors are returned as MCP tool errors.

```
  Profile     get_profile, update_profile
  Projects    list_projects, create_project, update_project, delete_project
  Experience  list_experience, create_experience, update_experience, delete_experience
  Skills      list_skills, create_skill, update_skill, delete_skill
  Posts       list_posts, get_post, create_post, update_post, delete_post,
              publish_post, unpublish_post, list_tags
  Posts flag  enable_posts, disable_posts
  SEO         get_seo_settings, update_seo_settings
  Content     export_content, import_content
```

Themes (Phase 6) and stats (Phase 7) tools are not available yet.

## Themes

The active theme is a site-wide setting (no visitor picker); visitors keep the light/dark toggle.
Every theme supplies a base token set plus light and dark overrides, and the base theme fills any
gap. The active theme is emitted as an inline `<style>` block, so there is no extra request.

Themes are edited at `/admin/themes` (or over `GET|POST /api/v1/admin/themes`,
`GET|PUT|DELETE /api/v1/admin/themes/{id}`, `POST /api/v1/admin/themes/{id}/activate`). The token
vocabulary covers colour pairs (`--card`, `--primary`, `--muted-foreground`, `--destructive`,
`--ring`, …), typography, shape/depth and code-highlighting tokens. Unknown token names and values
containing `; { } < >` are rejected, so a theme can never break out of the inline `<style>`.
The `johansen` base theme and the currently active theme cannot be deleted.

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
internal/db/repo_post.go  post, tag and post_tag repository
internal/db/repo_page_seo.go  page_seo per-route overrides
internal/db/repo_theme.go  theme CRUD with guard rails
internal/markdown/        goldmark + GFM + Chroma highlighting, sanitised
internal/mcp/             MCP streamable-HTTP server, auth, rate limiting and tools
internal/theme/           token vocabulary, defaults, validation, CSS emission
internal/icons/           vendored SVGs exposed to templates as inline SVG
internal/web/             chi router, handlers, auth, CSRF, content snapshot
internal/web/api_*.go     /api/v1 public reads and authenticated admin REST handlers
internal/web/posts.go     public post index, single post, tag archive
internal/web/feed.go      RSS 2.0 feed
internal/web/admin_posts.go  admin post CRUD and publish controls
internal/web/admin_seo.go    admin SEO settings and page overrides
internal/web/api_admin_seo.go  SEO settings REST handlers
internal/web/templates/   embedded html/template (public + admin)
internal/web/static/      embedded style.css, admin.css, fonts, htmx, favicons, avatar
```

## Tests

```sh
CGO_ENABLED=0 go test ./... -count=1
CGO_ENABLED=0 go vet ./...
gofmt -l .
```
