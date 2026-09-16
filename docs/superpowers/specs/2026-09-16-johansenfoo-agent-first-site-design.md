# johansenfoo — Agent-First Personal Site

**Status:** Approved
**Date:** 2026-09-16
**Supersedes:** none (replaces the static site in `mojoaar/johansenfoo-static`)

## Summary

Replace the current zero-build static landing page at `johansen.foo` with a single self-contained Go service whose content lives in SQLite and is editable through MCP, a REST API, and a small HTMX admin UI. The public page keeps its current design pixel-for-pixel. Content changes take effect on the next request — no rebuild, no HTML editing, and no way for the page and the `/me` JSON endpoint to drift apart.

## Goals

- Serve the existing public page from structured data rather than hand-edited HTML.
- Let an agent (or a human via `/admin`, or a script via curl) add and edit every piece of content.
- Expose public read APIs and an authenticated write API plus an MCP server.
- Add a Posts/Notes section with markdown, tags, hero images, drafts, and RSS.
- Support site-wide themes built from structured design tokens, shipping with a library seeded from real design systems.
- Be SEO-optimised with admin-configurable metadata, and mobile-first.
- Ship as one static binary in a multi-arch container image published to GHCR, deployed with docker compose behind the existing Nginx Proxy Manager.
- Success criterion: a content change made through MCP or `/admin` is visible on the live site on the next page load, with no rebuild and no file editing.

## Non-Goals (v1)

- Image uploads (images are referenced by URL or committed as static assets).
- Full-text search, comments, or multi-user accounts.
- Internationalisation — English only (explicitly dropped).
- A visitor-facing theme picker (the active theme is site-wide; visitors keep the light/dark toggle only).
- Changing the analytics provider.

## Background

The existing site (`/Users/mojoaar/Development/johansen_landing`) is a static page with no build step: `index.html` (structure plus inline vanilla JS), `style.css`, a `me` JSON profile duplicated byte-for-byte in `me.json`, favicons, `robots.txt`, `sitemap.xml`, and an `avatar.png`. It is served by an `nginx:alpine` container published to `ghcr.io/mojoaar/johansenfoo` and mapped to host port `8080`. TLS is terminated upstream by **Nginx Proxy Manager** — the container only ever speaks plain HTTP. Live responses confirm this: `server: openresty` with an `x-served-by: johansen.foo` header.

Two known defects this design eliminates:

1. `me` and `me.json` are two separate files that must be kept manually in sync forever. The new app generates `/me` from the database, so drift is structurally impossible.
2. The live `/me` does not return `Content-Type: application/json` despite the container's nginx configuration requesting it — the upstream proxy layer normalises the header. The new app sets it correctly itself.

The `nginx/default.conf`, `Dockerfile`, and static file list are retired with the old repo.

## Architecture

A single Go binary mirroring the conventions of `icloud-mailflow`:

```
cmd/johansenfoo/main.go        entry point, flags (-data=./data), version
internal/config/               JSON config (port, DB path, base URL)
internal/db/                   SQLite (modernc, no CGO) + migrations + repos
internal/markdown/             goldmark wrapper (GFM, sanitised)
internal/mcp/                  mcp-go server + API-key auth + rate limiting
internal/sysinfo/              cgroup / proc / Statfs readers for runtime stats
internal/theme/                token vocabulary, defaults, validation, CSS emission
internal/icons/                vendored SVGs, exposed to templates as inline SVG
internal/web/                  chi router, auth, handlers, prometheus, export/import
internal/web/templates/        embedded html/template (public + admin + docs)
internal/web/static/           embedded style.css, admin.css, code.css, fonts, htmx, favicons, avatar
```

**Stack:** Go 1.25, `github.com/go-chi/chi/v5`, `github.com/mark3labs/mcp-go`, `modernc.org/sqlite` (CGO-free), `html/template`, `github.com/yuin/goldmark`, `github.com/alecthomas/chroma/v2`, `github.com/prometheus/client_golang`, `github.com/catppuccin/go`.

**Rendering:** server-render from SQLite on each request, with a small in-memory snapshot cache invalidated on every write. Pre-rendering on write was considered and rejected — it adds moving parts for no benefit at this scale.

**Asset delivery:** no third-party origins on the critical path. Icons are rendered as inline SVG by Go from vendored sources; JetBrains Mono is self-hosted. The Lucide ES-module runtime and Font Awesome are removed entirely. Umami carries over unchanged.

## Data Model

SQLite, with migrations under `internal/db/migrations`. All timestamps are stored as ISO 8601 / RFC 3339 in UTC.

| Table | Columns |
| --- | --- |
| `profile` (singleton) | name, handle, location, dob, tagline, hero_bio, about_para_1, about_para_2, avatar |
| `social_link` | platform, url, label, sort, visible |
| `project` | name, url (nullable), description, icon (Lucide name), is_link, sort, visible |
| `experience` | years, role, company, icon, sort, visible |
| `skill` | name, sort |
| `post` | slug (unique), title, summary, body_md, status (`draft`\|`published`), published_at, created_at, updated_at, hero_image_url (nullable), hero_image_alt (nullable), seo_title, seo_description, og_image_url, canonical_url, noindex |
| `tag` | name, slug |
| `post_tag` | post_id, tag_id (many-to-many) |
| `page_seo` | route (unique), title, description, og_image_url, canonical_url, noindex |
| `theme` | slug (unique), name, description, tokens_light (JSON), tokens_dark (JSON), sort, created_at, updated_at |
| `page_view` | path, referrer (nullable), user_agent (nullable), ip_hash (nullable), created_at (indexed) |
| `settings` | key/value pairs including bcrypt password hash, API key, posts_enabled, active_theme, stats_enabled, stats_retention_days, timezone |

Theme token values are stored as JSON objects and validated server-side against a fixed vocabulary defined in `internal/theme`. Adding new token *values* never needs a migration; an unknown token *name* is rejected. The base theme `johansen` holds the complete token set; every other theme stores overrides only and inherits the remainder.

`settings` defaults: `posts_enabled=true`, `active_theme=johansen`, `stats_enabled=true`, `stats_retention_days=90`, `timezone=Europe/Copenhagen`.

### Content Seeding

A one-time migration seeds the database from the current `index.html` and `me`/`me.json` so day one is a byte-for-byte match: the profile, the 4 social links, the 9 projects in order (homelab as a non-link), the 8 experience rows, the 30 skills, and the `johansen` theme built from the existing CSS custom properties in `style.css`.

## Public Surface (unauthenticated)

| Route | Purpose |
| --- | --- |
| `GET /` | Rendered landing page |
| `GET /posts` | Post index, paginated |
| `GET /posts/{slug}` | Single published post |
| `GET /tags/{slug}` | Tag archive |
| `GET /feed.xml` | RSS 2.0 feed |
| `GET /sitemap.xml` | Generated live |
| `GET /robots.txt` | Generated from the configured body |
| `GET /me` | JSON profile, **generated from the database** |
| `GET /api/v1/profile` | Profile as JSON |
| `GET /api/v1/projects` | Projects as JSON |
| `GET /api/v1/experience` | Experience as JSON |
| `GET /api/v1/skills` | Skills as JSON |
| `GET /api/v1/posts` | Published posts, supports `?tag=` and pagination |
| `GET /api/v1/posts/{slug}` | Single published post |
| `GET /api/v1/theme` | Resolved token set as JSON |
| `GET /docs` | Self-documenting API and MCP reference |
| `GET /health` | Liveness |
| `GET /metrics` | Prometheus metrics |

**When `posts_enabled = false`:** `/posts`, `/posts/{slug}`, `/tags/{slug}`, `/api/v1/posts*`, and `/feed.xml` all return 404; the nav link is hidden; `/sitemap.xml` omits posts. Content is untouched in SQLite and returns when re-enabled.

## SEO

Metadata is resolved with clear precedence: per-page or per-post override → SEO defaults → hardcoded fallback.

**Global defaults** (admin-editable): site title, title template (e.g. `%s | johansen.foo`), default meta description, default OG image, OG type, Twitter card type, Twitter site handle, canonical base URL, `robots.txt` body, site-wide `noindex` toggle, sitemap toggle.

**Per-page overrides** live in `page_seo`, keyed by route: the home page (`/`), the post index (`/posts`), and tag archives (`/tags/{slug}`). The about section is part of the home page and therefore inherits the home page's metadata rather than having its own. **Per-post overrides** live on the `post` row: `seo_title`, `seo_description`, `og_image_url`, `canonical_url`, `noindex`. A post description defaults to its summary; a post OG image defaults to its hero image.

**Structured data** is generated server-side: a `Person` schema from the profile (as today) and a `BlogPosting` schema per post.

**Surfaces:** MCP tools `get_seo_settings` / `update_seo_settings`; REST `GET|PUT /api/v1/admin/settings/seo`; an admin form; documented on `/docs`.

## Mobile & Responsive

The current viewport meta and the 1→2→3 column breakpoints carry over unchanged. Additions:

- `srcset` / `sizes` on the avatar.
- `loading="lazy"` and `decoding="async"` on post hero images.
- Tap targets of at least 44px.
- `prefers-reduced-motion` honoured for all transitions.
- `-webkit-text-size-adjust: 100%`.
- `theme-color` becomes theme-driven (today it is hardcoded `#0f1117` / `#f4f6fb`), so mobile browser chrome follows the active theme.

Verified with a Lighthouse mobile pass; the resulting Core Web Vitals work feeds the SEO effort.

## Authenticated Surface

Authentication is a session cookie (from the bcrypt admin password) **or** `Authorization: Bearer <api-key>`.

**Admin UI** — `/admin` and `/admin/*`, HTMX-driven CRUD for every entity, publish/unpublish, API-key regeneration, password change, a theme list with a token editor and "set active theme", and stats on/off plus retention settings.

**Admin dashboard** — visitor cards (views today / 7d / 30d, daily uniques, top pages, top referrers, recent hits) and runtime cards. Runtime cards show goroutines, heap, GC and uptime from the Prometheus collectors, plus container CPU percentage, memory used against limit, and disk used against total for the data volume. The runtime panel polls over HTMX roughly every 5 seconds; visitor cards refresh on load.

**Admin REST API:**
- `/api/v1/admin/profile`, `/projects`, `/experience`, `/skills`, `/posts` — full CRUD
- `/api/v1/admin/themes` — list, get, create, update, delete
- `/api/v1/admin/settings/seo` — `GET|PUT`
- `/api/v1/admin/settings/posts` — `PUT {"enabled": bool}`
- `/api/v1/admin/settings/theme` — `PUT`
- `/api/v1/admin/export` — `GET`, whole-content JSON snapshot
- `/api/v1/admin/import` — `POST`, restore from snapshot
- `/api/v1/admin/stats/visitors?period=7d` — `GET`
- `/api/v1/admin/stats/system` — `GET`

## MCP Tools

Served over streamable HTTP at `/mcp`. Typed schemas, agent-oriented descriptions, snake_case names, errors returned via `mcp.NewToolResultError`.

- Profile: `get_profile`, `update_profile`
- Projects: `list_projects`, `create_project`, `update_project`, `delete_project`
- Experience: `list_experience`, `create_experience`, `update_experience`, `delete_experience`
- Skills: `list_skills`, `create_skill`, `update_skill`, `delete_skill`
- Posts: `list_posts`, `get_post`, `create_post`, `update_post`, `delete_post`, `publish_post`, `unpublish_post`
- Tags: `list_tags`
- Posts toggle: `enable_posts`, `disable_posts` — explicit paired tools, never a boolean flag argument
- Themes: `list_themes`, `get_theme`, `create_theme`, `update_theme`, `delete_theme`, `set_active_theme`, `import_theme`
- SEO: `get_seo_settings`, `update_seo_settings`
- Stats: `get_visitor_stats`, `get_system_stats`, `clear_visitor_stats`
- Content: `export_content`, `import_content`

`create_post` and `update_post` accept `tags` (array), `hero_image_url`, and `hero_image_alt`. `get_theme` returns resolved tokens with inheritance applied; `update_theme` merges partial token updates server-side.

**Guard rails:** the base `johansen` theme cannot be deleted, and the currently active theme cannot be deleted.

## Theming

The active theme is a site-wide setting; there is no visitor-facing theme picker. Visitors keep the existing light/dark toggle, which remains a separate axis: `<html data-theme="johansen" data-mode="dark">`. `data-mode` is still persisted in `localStorage` and still initialised from `prefers-color-scheme` exactly as the current inline IIFE does.

Every theme supplies both a light and a dark token set; the base theme fills any gaps. The active theme is emitted as an inline `<style>` block in `<head>` — no extra request and no flash of unstyled content.

**Token vocabulary** (widened beyond the original thin set):

- **Colour** — the full shadcn-style surface/foreground pair set: `background`/`foreground`, `card`, `popover`, `primary`, `secondary`, `muted`, `accent`, `destructive`, `border`, `input`, `ring`. Chart tokens are deliberately omitted — the site renders no charts.
- **Typography** — font pairings, type scale, line-height, letter-spacing, weights.
- **Shape & depth** — radius, border width, shadow set.
- **Syntax highlighting** — code tokens derived from the theme palette.

**Seed library:** Catppuccin's four flavours (Latte, Frappé, Macchiato, Mocha) via `github.com/catppuccin/go`, plus Nord, Rosé Pine, Tokyo Night, Gruvbox, Everforest, and Solarized, all of which already publish CSS-variable palettes.

`import_theme` ingests a shadcn- or Catppuccin-format palette and creates a theme, so the library grows without a code change or rebuild.

**Constraint:** a token theme varies colour, type, shape, and depth — not layout.

## Posts

- **Tags** — multi-value, shown as chips on the post and in the list, filterable with `?tag=`, with a `/tags/{slug}` archive page.
- **Hero image** — a URL or a committed static asset, no uploads in v1, used as the per-post OG image by default.
- **Dates** — stored as RFC 3339 UTC. Rendered as ISO 8601 on a **24-hour clock** in the site timezone for the UI (`2026-09-16 14:30`) and in the API (`2026-09-16T14:30:00+02:00`). The RSS `pubDate` is emitted as **RFC 1123**, which RSS 2.0 requires — ISO 8601 breaks readers.
- **Drafts** — `status = draft` hides a post from the public site and the public API. Independent of the global `posts_enabled` kill switch.

## Markdown & Syntax Highlighting

Markdown is parsed with goldmark (GFM) and sanitised before rendering. Code highlighting is performed **server-side with Chroma** through goldmark's highlighting extension, emitting CSS *classes* rather than inline styles. Two stylesheets are generated at startup and served as `/static/code.css`, scoped so they follow the theme, meaning highlighting works with no client JavaScript. The Chroma theme ultimately derives from the active theme's palette, so code blocks match the site — the Catppuccin synergy.

## Visitor Stats & Privacy

A page-view middleware records GET requests that return HTML on public routes only. It explicitly **excludes `/admin`, `/static`, `/api`, `/mcp`, and `/metrics`**.

No cookies, no raw IPs, no PII: an IP is stored only as a salted hash, and the **salt rotates daily** so a visitor cannot be correlated across days. Because the salt rotates, uniqueness can only be computed within a single day — the 7-day and 30-day figures on the dashboard are therefore **sums of daily uniques**, not deduplicated visitors across the whole window. A daily background goroutine prunes `page_view` rows older than `stats_retention_days`. `stats_enabled` turns collection off entirely.

## Runtime Stats

Go process figures — goroutines, heap, GC, uptime — come free from the existing Prometheus collectors. Host and container CPU, memory, and disk come from `internal/sysinfo/`, which reads cgroup limits (`/sys/fs/cgroup/cpu.max`, `memory.max`, `memory.current`) with a `/proc/*` fallback, and `syscall.Statfs` on the data directory. No new dependency, container-aware, and it degrades to "unavailable" off Linux so macOS local development does not crash. The same figures remain available at `/metrics`.

## Auth & Security

- bcrypt admin password; a session cookie for `/admin`.
- A server-generated, regenerable API key, shown once, used as a Bearer token for MCP and the REST API.
- CSRF protection on stateful POSTs: skipped for GET/HEAD, for requests with `HX-Request: true`, and for `/login`, `/setup`, and `/mcp`. Cookie name `johansenfoo_csrf`, `SameSite=Strict`, `HttpOnly`, `Secure` when behind TLS.
- Rate limiting on login (10 attempts per 15 minutes per IP) and on MCP (100 requests per minute per IP).
- Security headers and a 30-second request timeout.

**Open question:** the date of birth stays exposed in `/me` for parity with the current endpoint. This is flagged in case it should be dropped instead.

## Deployment & CI

- **Dockerfile** — multi-stage: `golang:alpine` builder (`CGO_ENABLED=0`, `-ldflags="-s -w"`) → `alpine` plus `ca-certificates`, `EXPOSE 8080`, `VOLUME ["/data"]`, entrypoint `-data=/data`.
- **docker-compose.yml** — `ghcr.io/mojoaar/johansenfoo:latest`, `8080:8080`, `./data:/data`, `restart: unless-stopped`, `TZ=Europe/Copenhagen`.
- **`.github/workflows/docker.yml`** — buildx, multi-arch `linux/amd64,linux/arm64`, on push to `main` and `v*` tags.
- **`.github/workflows/test.yml`** — `go test ./...` and `go vet ./...`. Two separate workflows, matching `icloud-mailflow`.
- **Nginx Proxy Manager keeps terminating TLS and proxying `johansen.foo` to host `:8080`. No proxy change is required.**
- Release flow, versioning (version held in both `cmd/johansenfoo/main.go` and the `VERSION` file), Keep a Changelog, `AGENTS.md`, `README.md`, and an MIT licence all follow the existing conventions.

### GitHub Migration

The new repo takes the name `mojoaar/johansenfoo`; the existing static repo is renamed to `mojoaar/johansenfoo-static` and archived.

**Ordering matters:** rename and archive the old repo **before** the new workflow's first push. The old repo's workflow publishes the GHCR package `ghcr.io/mojoaar/johansenfoo`, so the package name must be freed or the two collide. After the first publish, the new package must also be set public (`gh api --method PATCH /user/packages/container/johansenfoo/visibility -f visibility=public`) so any host can pull it without authentication.

### Mandatory documentation updates

Following the `icloud-mailflow` convention: a new feature, endpoint, or MCP tool requires updating the `/docs` reference and its sidebar; a new feature also updates the README feature list; a new package updates the README architecture tree and the `AGENTS.md` architecture section; changed or removed routes update both `/docs` and the README.

## Testing

Table-driven `_test.go` files per package covering the database repos, handlers, auth, MCP tools, markdown, theme, and sysinfo. Routes are exercised with `httptest` against a temporary or in-memory SQLite database. `go test ./...` and `go vet ./...` must pass.

## Implementation Phases

The spec describes one product, but it is too much for a single uninterrupted build. Implementation is phased, each phase leaving the site in a working state:

1. **Foundation** — repo scaffold, config, database with migrations and content seeding, the `theme` package with the token vocabulary and the `johansen` theme, the `icons` package, the landing page ported to `html/template` and rendered from the database, `/me`, static assets, `/health`. *Deliverable: a visually identical public site that already reads its content from SQLite.*
2. **Auth & admin** — bcrypt password, setup and login, session cookie, CSRF, and the HTMX admin CRUD UI for profile, projects, experience, and skills.
3. **REST API** — the public read endpoints and the authenticated write endpoints, plus export and import.
4. **Posts** — the markdown and Chroma pipeline, `code.css`, posts CRUD, tags, drafts, RSS, the `posts_enabled` kill switch, and the SEO settings and `page_seo` overrides.
5. **MCP** — the mcp-go server at `/mcp`, API-key auth, rate limiting, and the full tool set.
6. **Theme library** — the Catppuccin, Nord, Rosé Pine, Tokyo Night, Gruvbox, Everforest, and Solarized seed themes, the theme token editor in admin, and `import_theme`.
7. **Stats** — the page-view middleware, visitor statistics, `sysinfo` runtime statistics, the dashboard panels, and the retention job.
8. **Ops** — Dockerfile, docker-compose, both GitHub Actions workflows, the `/docs` reference, and README, AGENTS, CHANGELOG, and VERSION.

## Open Questions

1. Should the date of birth remain in the public `/me` response, as it is today?
2. Should the Umami snippet be dropped now that built-in visitor stats exist? The working assumption is to keep it untouched, which means traffic is counted twice.
