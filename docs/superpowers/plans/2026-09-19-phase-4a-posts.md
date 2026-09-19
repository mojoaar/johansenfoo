# Phase 4a — Posts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a complete Posts system — markdown posts with server-side Chroma highlighting, tags, drafts, hero images, pagination, an RSS feed, a `posts_enabled` kill switch, an admin CRUD UI, and public/admin REST endpoints — so an author can publish a post that is immediately live on the public site.

**Architecture:** All changes live in `internal/db` (one append-only migration plus repositories), `internal/markdown` (Chroma), `internal/web` (public pages, admin UI, REST), and `internal/web/templates` + `static`. Public reads project the existing `ContentStore` for profile/theme, but posts are read from their own repository because their cardinality and pagination do not belong in the snapshot; the snapshot still drives navigation (whether the nav link shows) via `posts_enabled`. Every admin write calls `d.Content.Reload()`.

**Tech Stack:** Go 1.25, `github.com/go-chi/chi/v5`, `modernc.org/sqlite` (CGO-free), `github.com/yuin/goldmark` + `github.com/yuin/goldmark-highlighting/v2` + `github.com/alecthomas/chroma/v2` (classes, no inline styles), `html/template`, vendored HTMX.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Data Model `post`/`tag`/`post_tag`; Public Surface; Posts; Markdown & Syntax Highlighting; Implementation Phases §4 — posts half).

## Global Constraints

Copied verbatim from the approved spec and earlier plans. Every task's requirements implicitly include this section.

- Module path: `github.com/mojoaar/johansenfoo`. Go directive: `go 1.25.5`.
- No CGO anywhere: every command runs as `CGO_ENABLED=0 go test ./...` / `CGO_ENABLED=0 go build ./...`.
- No comments in Go code unless essential. `//go:embed` is the only sanctioned exception.
- No third-party origin on the runtime critical path except the Umami snippet in `base.html`. HTMX stays vendored; Chroma runs server-side and emits CSS classes.
- All timestamps are stored as RFC 3339 UTC via `strftime('%Y-%m-%dT%H:%M:%SZ','now')`, never `datetime('now')`.
- All SQL is parameterized.
- `<html>` carries `data-theme="<slug>"` and `data-mode="light|dark"` as separate axes.
- Migrations are append-only: `0001`-`0005` stay byte-identical. **Phase 4a adds migration `0006`.**
- Every task ends with `CGO_ENABLED=0 go test ./... -count=1`, `CGO_ENABLED=0 go vet ./...` and `gofmt -l .` clean, then a commit.
- Documentation is part of the definition of done.
- Status enum is exactly `draft` | `published`. Drafts are hidden from every public surface.

## Scope of this phase

In scope: the `post`, `tag`, `post_tag` tables; markdown + Chroma; `/posts`, `/posts/{slug}`, `/tags/{slug}`; pagination; RSS; `posts_enabled`; admin post CRUD; `/api/v1/posts*` and `/api/v1/admin/posts*`.

Out of scope (Phase 4b / later, do not stub): `page_seo`, global SEO settings editing, per-post SEO overrides beyond rendering defaults, `BlogPosting` schema, MCP tools, stats. The `post` table carries its SEO columns from day one (they are in the spec's data model and the migration is append-only), but 4a only reads `seo_title`/`seo_description`/`og_image_url`/`canonical_url`/`noindex` via the existing `resolveMeta` defaults; full override precedence and `page_seo` land in 4b.

## Review Focus

Failure modes the spec implies that the tasks below must pin:

1. **Drafts and hidden content leaking publicly.** Every public posts surface (`/posts`, `/posts/{slug}`, `/tags/{slug}`, `/feed.xml`, `/api/v1/posts*`, `sitemap.xml`) must exclude `status = 'draft'`. A draft fetched by slug must be 404, not 200.
2. **Markdown XSS.** `body_md` is rendered with goldmark + Chroma and injected as `template.HTML`; raw HTML and `javascript:` URLs in a post must be neutralised.
3. **`posts_enabled=false` behaving like deletion.** With the switch off, `/posts`, `/posts/{slug}`, `/tags/{slug}`, `/api/v1/posts*`, `/feed.xml` return 404, the nav link is hidden, and sitemap omits posts — but posts remain in SQLite and return when re-enabled.
4. **Timestamps/timezone drift.** Stored UTC; UI renders the site timezone on a 24-hour clock; the API emits an offset; RSS emits RFC 1123 (ISO 8601 breaks readers).
5. **Slug collisions and injection.** Duplicate slug and duplicate tag must be handled deterministically; tag input is normalised to a slug; a slug path segment cannot traverse.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/db/migrations/0006_posts.sql` | `post`, `tag`, `post_tag` tables and indexes |
| `internal/db/models.go` | Modified: `Post`, `Tag` types with JSON tags |
| `internal/db/repo_post.go` | `PostRepo`: list/get/create/update/delete/publish, tag assignment, tag queries |
| `internal/db/repo_post_test.go` | Repository tests |
| `internal/markdown/markdown.go` | Modified: goldmark + GFM + Chroma highlighting (classes), sanitised |
| `internal/markdown/markdown_test.go` | Rendering and XSS tests |
| `internal/web/static/code.css` | Chroma class → theme-token stylesheet (no JS) |
| `internal/web/posts.go` | Public post index, single post, tag archive, pagination, date helpers |
| `internal/web/posts_test.go` | Public page tests |
| `internal/web/feed.go` | RSS 2.0 feed |
| `internal/web/feed_test.go` | Feed tests (RFC 1123) |
| `internal/web/admin_posts.go` | Admin post CRUD handlers |
| `internal/web/admin_posts_test.go` | Admin tests |
| `internal/web/api_posts.go` | Public `/api/v1/posts*` handlers |
| `internal/web/api_admin_posts.go` | Admin `/api/v1/admin/posts*` handlers |
| `internal/web/api_posts_test.go` | REST tests |
| `internal/web/templates/posts.html` | Post index |
| `internal/web/templates/post.html` | Single post |
| `internal/web/templates/tag.html` | Tag archive |
| `internal/web/templates/admin_posts.html` | Admin post list |
| `internal/web/templates/admin_post_form.html` | Admin post create/edit |
| `internal/web/templates/landing.html` | Modified: Posts nav link (conditional) |
| `internal/web/templates/base.html` | Modified: link `/static/code.css` |
| `internal/web/templates/base_post.html` | Public shell for post pages (nav + footer) |
| `internal/web/web.go` | Modified: register public posts, feed, admin, API routes |
| `internal/web/seo.go` | Modified: sitemap includes published posts and tag pages |
| `internal/db/backup.go` | Modified: include posts/tags in export/import |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Modified |

---

### Task 1: Schema, models, and the post repository

**Files:**
- Create: `internal/db/migrations/0006_posts.sql`
- Create: `internal/db/repo_post.go`
- Create: `internal/db/repo_post_test.go`
- Modify: `internal/db/models.go`

**Interfaces:**
- Produces:
  - `db.Post{ID int64; Slug, Title, Summary, BodyMD, Status string; PublishedAt *time.Time; CreatedAt, UpdatedAt time.Time; HeroImageURL, HeroImageAlt, SEOTitle, SEODescription, OGImageURL, CanonicalURL string; NoIndex bool; Tags []Tag}`
  - `db.Tag{ID int64; Name, Slug string}`
  - `db.ErrPostNotFound = errors.New("post not found")`
  - `db.NewPostRepo(*sql.DB) *PostRepo`
  - `(*PostRepo).Published(limit, offset int) ([]Post, error)`, `CountPublished() (int, error)`
  - `(*PostRepo).ByTag(tagSlug string, limit, offset int) ([]Post, error)`, `CountByTag(tagSlug string) (int, error)`
  - `(*PostRepo).PublishedBySlug(slug string) (*Post, error)`
  - `(*PostRepo).All() ([]Post, error)`, `ByID(id int64) (*Post, error)`
  - `(*PostRepo).Create(*Post) (int64, error)`, `Update(*Post) error`, `Delete(id int64) error`, `SetStatus(id int64, status string) error`
  - `(*PostRepo).Tags() ([]Tag, error)`, `TagBySlug(slug string) (*Tag, error)`

- [ ] **Step 1: Write the failing tests**

Create `internal/db/repo_post_test.go` covering: create with tags round-trips; `Published` excludes drafts and orders by `published_at DESC`; `PublishedBySlug` rejects a draft with `ErrPostNotFound`; `ByTag` returns only that tag's published posts; `CountPublished` matches; `Update` replaces tags; `SetStatus` publishes; `Delete` removes the post and its `post_tag` rows.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestPost -v`
Expected: FAIL — `PostRepo` and the tables do not exist.

- [ ] **Step 3: Write migration 0006**

Create `internal/db/migrations/0006_posts.sql`:

```sql
CREATE TABLE IF NOT EXISTS post (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    slug            TEXT NOT NULL UNIQUE,
    title           TEXT NOT NULL,
    summary         TEXT NOT NULL DEFAULT '',
    body_md         TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    hero_image_url  TEXT,
    hero_image_alt  TEXT,
    seo_title       TEXT NOT NULL DEFAULT '',
    seo_description TEXT NOT NULL DEFAULT '',
    og_image_url    TEXT NOT NULL DEFAULT '',
    canonical_url   TEXT NOT NULL DEFAULT '',
    noindex         INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_post_status_published_at ON post (status, published_at DESC);

CREATE TABLE IF NOT EXISTS tag (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS post_tag (
    post_id INTEGER NOT NULL REFERENCES post (id) ON DELETE CASCADE,
    tag_id  INTEGER NOT NULL REFERENCES tag (id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);
```

- [ ] **Step 4: Add the models**

In `internal/db/models.go`, add:

```go
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Post struct {
	ID             int64      `json:"id"`
	Slug           string     `json:"slug"`
	Title          string     `json:"title"`
	Summary        string     `json:"summary"`
	BodyMD         string     `json:"body_md"`
	Status         string     `json:"status"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	HeroImageURL   string     `json:"hero_image_url"`
	HeroImageAlt   string     `json:"hero_image_alt"`
	SEOTitle       string     `json:"seo_title"`
	SEODescription string     `json:"seo_description"`
	OGImageURL     string     `json:"og_image_url"`
	CanonicalURL   string     `json:"canonical_url"`
	NoIndex        bool       `json:"noindex"`
	Tags           []Tag      `json:"tags"`
}
```

Add `"time"` to `models.go` imports.

- [ ] **Step 5: Write the repository**

Create `internal/db/repo_post.go`. Key implementation points:

- A private `scanPost` reading the 16 scalar columns, `published_at` through `sql.NullString`, and parsing RFC 3339; `created_at`/`updated_at` via `strftime('%Y-%m-%dT%H:%M:%SZ','now')` on every write.
- `Create` inserts the post inside a transaction, then for each tag slug upserts the tag (`INSERT ... ON CONFLICT(slug) DO UPDATE SET name = excluded.name RETURNING id`) and inserts `post_tag`; commit.
- `Update` updates the scalar columns, then `DELETE FROM post_tag WHERE post_id = ?` and re-inserts the tag set, in the same transaction.
- `Delete` relies on `ON DELETE CASCADE` — the DSN enables `foreign_keys=1`, so the `post_tag` rows go automatically.
- Tag loading: `tagsFor(postID)` and a bulk `attachTags([]Post)` that does one query joining `post_tag`/`tag` ordered by `tag.name`, grouped by post id.
- `Published(limit, offset)` uses `WHERE status = 'published' AND published_at IS NOT NULL ORDER BY published_at DESC, id DESC LIMIT ? OFFSET ?`.
- `PublishedBySlug(slug)` uses `WHERE slug = ? AND status = 'published'`; a miss (including a draft) returns `ErrPostNotFound`.
- `SetStatus(id, "published")` sets `status` and `published_at = COALESCE(published_at, strftime('%Y-%m-%dT%H:%M:%SZ','now'))`; setting `draft` leaves `published_at` intact (so re-publishing keeps the original date).

- [ ] **Step 6: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestPost -v`
Expected: PASS. Existing `TestMigrateIsIdempotent` and `TestSchemaHasCoreTables` still pass; extend the latter with `post`, `tag`, `post_tag`.

- [ ] **Step 7: Full gate and commit**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`

```bash
git add internal/db/migrations/0006_posts.sql internal/db/models.go internal/db/repo_post.go internal/db/repo_post_test.go internal/db/db_test.go
git commit -m "feat: add posts schema and repository"
```

---

### Task 2: Markdown + Chroma pipeline and `code.css`

**Files:**
- Modify: `internal/markdown/markdown.go`
- Create: `internal/markdown/markdown_test.go`
- Create: `internal/web/static/code.css`
- Modify: `internal/web/templates/base.html`

**Interfaces:**
- Produces: `markdown.Render(src string) template.HTML` now highlights fenced code with classes; `markdown.Plain(src string) string` optionally strips to text (not required).
- Consumes: nothing from Task 1.

- [ ] **Step 1: Add the dependencies**

Run:

```sh
go get github.com/yuin/goldmark-highlighting/v2@latest github.com/alecthomas/chroma/v2@latest
```

Confirm `go.mod` still says `go 1.25.5` (the last `go get` in Phase 2 tried to bump it; pin if needed) and that `CGO_ENABLED=0 go build ./...` works.

- [ ] **Step 2: Write the failing tests**

Create `internal/markdown/markdown_test.go`:

- `TestRenderHighlightsCodeWithClasses`: input a fenced ` ```go ` block containing `func main() {}`; assert the output contains `class="chroma"` and a `<span class="k">` (keyword) and does **not** contain `style="`.
- `TestRenderEscapesRawHTML`: input `<script>alert(1)</script>`; assert the output HTML-escapes it (no live `<script>` element). This requires `goldmark.WithRendererOptions(html.WithUnsafe())` to be OFF (the default), so raw HTML is escaped.
- `TestRenderSanitisesJavascriptURL`: input `[x](javascript:alert(1))`; assert the output anchor `href` is not `javascript:`.
- `TestRenderGFM`: input a table and a strikethrough; assert `<table` and `<del>`.

- [ ] **Step 3: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/markdown/ -v`
Expected: FAIL — no highlighting extension, no `<span class="k">`.

- [ ] **Step 4: Modify the renderer**

Replace the body of `internal/markdown/markdown.go`:

```go
package markdown

import (
	"bytes"
	"html/template"

	"github.com/alecthomas/chroma/v2/styles"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("catppuccin-mocha"),
			highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
		),
	),
)

func Render(src string) template.HTML {
	if src == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(src))
	}
	return template.HTML(buf.String())
}

var _ = styles.Get
```

The `styles.Get` reference ensures the style token package stays imported; drop it and the `styles` import if it is unused after `go mod tidy` (use `go mod tidy` to settle imports).

- [ ] **Step 5: Write `code.css`**

Create `internal/web/static/code.css` mapping Chroma's stable classes to the theme's existing tokens so highlighting follows `data-mode` with no JavaScript and no per-theme generation. Map at least: `.chroma` (background/foreground), `.chroma .k`/`.kd`/`.kn`/`.kr`/`.kt` (keywords → `--accent`), `.chroma .s`/`.s1`/`.s2`/`.sb`/`.sc` (strings → `--green`), `.chroma .c`/`.c1`/`.cm` (comments → `--text-muted`), `.chroma .nf`/`.fm` (functions → `--accent2`), `.chroma .mi`/`.m` (numbers), `.chroma .o` (operators). Do not hardcode hex colours; use `var(--...)` so the theme and mode drive it.

- [ ] **Step 6: Link the stylesheet**

In `base.html`, add after the `style.css` link: `<link rel="stylesheet" href="/static/code.css" />`.

- [ ] **Step 7: Run the tests and full gate**

Run: `CGO_ENABLED=0 go test ./internal/markdown/ -v && CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: markdown tests pass; the wider suite stays green (no existing test renders a fenced block except possibly landing markup tests — if one does, update it to expect the chroma wrapper).

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum internal/markdown internal/web/static/code.css internal/web/templates/base.html
git commit -m "feat: render markdown code blocks with chroma classes"
```

---

### Task 3: Public post index and single post

**Files:**
- Create: `internal/web/posts.go`
- Create: `internal/web/posts_test.go`
- Create: `internal/web/templates/base_post.html`
- Create: `internal/web/templates/posts.html`
- Create: `internal/web/templates/post.html`
- Modify: `internal/web/templates/landing.html` (nav link)
- Modify: `internal/web/render.go` (add post fields to `page`)
- Modify: `internal/web/web.go` (routes)

**Interfaces:**
- Consumes: `db.NewPostRepo(d.DB)`, `markdown.Render`, `resolveMeta`, `theme.CSS`, `newPage`.
- Produces:
  - `postsEnabled(c *db.SiteContent) bool` — reads `c.Settings["posts_enabled"]`, default true.
  - `formatPostDate(t time.Time, loc *time.Location) string` — `2006-01-02 15:04` (24h) in the site timezone.
  - `siteLocation(c *db.SiteContent) *time.Location` — `time.LoadLocation(c.Settings["timezone"])`, fallback `time.UTC`.
  - `apiPostBase`/template data: `page` gains `Posts []db.Post`, `Post db.Post`, `PostBody template.HTML`, `PageNum`, `TotalPages int`, `Tags []db.Tag`, `Tag db.Tag`, `PostsEnabled bool`, `CodeCSS bool`.
  - `postIndexHandler(d Deps) http.HandlerFunc`, `postHandler(d Deps) http.HandlerFunc`.

- [ ] **Step 1: Write the failing tests**

`internal/web/posts_test.go`:
- `TestPostsIndexListsPublished`: publish two posts (one draft) via the repo; `GET /posts` returns 200, contains the two published titles, not the draft title.
- `TestPostPageRendersMarkdown`: publish a post whose body is `# Hi\n\n```go\nfunc main() {}\n````; `GET /posts/{slug}` returns 200 and contains `<h1` and `class="chroma"`.
- `TestDraftPostIs404`: a draft's slug returns 404.
- `TestMissingPostIs404`.
- `TestNavShowsPostsWhenEnabledAndHidesWhenDisabled`: `GET /` contains `href="/posts"` with `posts_enabled=true`; after setting it `false` and reloading, it does not.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestPostsIndex|TestPostPage|TestDraftPost|TestMissingPost|TestNavShows' -v`
Expected: FAIL — routes 404 / `postsEnabled` undefined.

- [ ] **Step 3: Add the public shell template**

Create `base_post.html`, modelled on the landing nav and footer but with the nav linking to `/` (`#projects`, `#about` become `/`-anchored: use `/#projects`, `/#about`) and a conditional Posts link when `.PostsEnabled`. Render the theme tokens and `code.css`. Keep the same theme-toggle IIFE and hamburger script.

- [ ] **Step 4: Write the handlers**

In `internal/web/posts.go` implement:
- `postsEnabled`, `siteLocation`, `formatPostDate`.
- `postIndexHandler`: if `!postsEnabled` → 404. Read `?page=` (default 1, min 1) and `?tag=`; page size 10. `CountPublished`/`CountByTag` → `TotalPages`. Load posts, `attachTags` already done by repo, render `posts.html` on `base_post.html`.
- `postHandler`: if `!postsEnabled` → 404. `PublishedBySlug(chi.URLParam(r,"slug"))`; on `ErrPostNotFound` → 404. Render `post.html` with `PostBody = markdown.Render(post.BodyMD)`.

Register in `web.go` before `/admin`: `r.Get("/posts", postIndexHandler(d))`, `r.Get("/posts/{slug}", postHandler(d))`.

- [ ] **Step 5: Add the nav link**

In `landing.html` nav, add `{{if .PostsEnabled}}<li><a href="/posts">Posts</a></li>{{end}}` as the first list item after Home. Set `data.PostsEnabled = postsEnabled(c)` in `newPage`? No — `newPage` is also used by posts pages. Set it in `landingHandler` (and in the posts handlers) from the content snapshot. The post pages use `base_post.html`, which also reads `.PostsEnabled`.

- [ ] **Step 6: Write the templates**

`posts.html` iterates `.Posts`: title link, `formatPostDate(.PublishedAt)`, summary, tag chips linking `/tags/{slug}`, and pagination links. `post.html` renders title, date, hero image when `HeroImageURL` is set, `.PostBody`, and the tag chips. Both use `{{if .HeroImageURL}}`.

- [ ] **Step 7: Run the tests and full gate**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestPosts|TestPost|TestDraft|TestMissing|TestNavShows' -v && CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/web/posts.go internal/web/posts_test.go internal/web/templates/base_post.html internal/web/templates/posts.html internal/web/templates/post.html internal/web/templates/landing.html internal/web/render.go internal/web/web.go
git commit -m "feat: render public post index and single post pages"
```

---

### Task 4: Tag archive and filtering

**Files:**
- Modify: `internal/web/posts.go`
- Modify: `internal/web/posts_test.go`
- Create: `internal/web/templates/tag.html`
- Modify: `internal/web/web.go`

**Interfaces:**
- Consumes: `db.NewPostRepo(d.DB).ByTag/CountByTag/TagBySlug`.
- Produces: `tagArchiveHandler(d Deps) http.HandlerFunc`.

- [ ] **Step 1: Write the failing tests**

- `TestTagArchiveListsOnlyThatTag`: two published posts with different tags; `GET /tags/{slug}` returns 200, contains one title and not the other.
- `TestUnknownTagIs404`.
- `TestPostsIndexFiltersByTag`: `GET /posts?tag={slug}` returns only that tag's posts.
- `TestTagChipsLinkToArchive`: the index and single post contain `href="/tags/{slug}"`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestTagArchive|TestUnknownTag|TestPostsIndexFilters|TestTagChips' -v`
Expected: FAIL — no `/tags/{slug}` route.

- [ ] **Step 3: Implement**

`tagArchiveHandler`: if `!postsEnabled` → 404. `TagBySlug`; miss → 404. Paginate `ByTag`. Render `tag.html` with the tag name and the same post-list partial as `posts.html` (use a shared `{{define "post_list"}}` template block parsed once).

Register `r.Get("/tags/{slug}", tagArchiveHandler(d))`.

- [ ] **Step 4: Run the tests and full gate, then commit**

```bash
git add internal/web/posts.go internal/web/posts_test.go internal/web/templates/tag.html internal/web/templates/posts.html internal/web/web.go
git commit -m "feat: add tag archives and post tag filtering"
```

---

### Task 5: RSS feed

**Files:**
- Create: `internal/web/feed.go`
- Create: `internal/web/feed_test.go`
- Modify: `internal/web/web.go`

**Interfaces:**
- Consumes: `PostRepo.Published`, `siteLocation`, `c.Settings["canonical_base_url"]`, `c.Profile`.
- Produces: `feedHandler(d Deps) http.HandlerFunc`; `formatRSSDate(t time.Time) string` = `t.UTC().Format(time.RFC1123Z)`.

- [ ] **Step 1: Write the failing tests**

- `TestFeedIsValidRSS`: `GET /feed.xml` returns 200, `Content-Type: application/rss+xml; charset=utf-8`, contains `<rss version="2.0"`, `<channel>`, and each published post's `<item>`.
- `TestFeedExcludesDrafts`: a draft title is absent.
- `TestFeedPubDateIsRFC1123`: the `<pubDate>` value parses with `time.RFC1123Z` and matches the post's `published_at` converted to UTC.
- `TestFeedDisabledWhenPostsDisabled`: with `posts_enabled=false`, `GET /feed.xml` → 404.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestFeed -v`
Expected: FAIL — 404, no route.

- [ ] **Step 3: Implement**

`feedHandler` writes RSS 2.0 manually with `encoding/xml` or a `bytes.Buffer` (XML-escaping each field with `xml.EscapeText`). Channel: title from `c.Profile.Name + " — Blog"`, link `base + "/posts"`, description from `settings["seo_description"]`, `lastBuildDate` RFC 1123. Items: title, link `base + "/posts/" + slug`, `<guid isPermaLink="true">` the link, `<pubDate>` RFC 1123, `<description>` the summary. Register `r.Get("/feed.xml", feedHandler(d))`.

- [ ] **Step 4: Run the tests and full gate, then commit**

```bash
git add internal/web/feed.go internal/web/feed_test.go internal/web/web.go
git commit -m "feat: add the RSS feed"
```

---

### Task 6: The `posts_enabled` kill switch and sitemap

**Files:**
- Modify: `internal/web/seo.go`
- Modify: `internal/web/seo_test.go`
- Modify: `internal/web/web.go` (only if routes need guarding — prefer guarding in each handler)

**Interfaces:**
- Consumes: `postsEnabled`, `PostRepo.Published`, `PostRepo.Tags`.
- Produces: sitemap includes `/posts` and each published post and each tag archive only when posts are enabled; omits all when disabled. A helper `postsKillSwitch(next http.HandlerFunc, enabled func() bool)` is **not** used — each handler checks `postsEnabled` itself so the 404 shape is identical.

- [ ] **Step 1: Write the failing tests**

- `TestPostsKillSwitch`: set `posts_enabled=false`, reload; assert `/posts`, `/posts/{slug}`, `/tags/{slug}`, `/feed.xml`, `/api/v1/posts` all return 404 (the API one may land in Task 9; guard it there too and assert here once it exists).
- `TestSitemapIncludesPublishedPostsWhenEnabled`: contains `{base}/posts/{slug}` for a published post and `{base}/posts`.
- `TestSitemapOmitsPostsWhenDisabled`: with the switch off, no `/posts` URLs.
- `TestSitemapExcludesDrafts`: a draft slug never appears.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestPostsKillSwitch|TestSitemap' -v`
Expected: FAIL — sitemap has only `/`.

- [ ] **Step 3: Implement**

Extend `sitemapHandler`: after the root URL, if `postsEnabled(c)`, add `/posts`, each published post URL, and each tag archive. Also honour `settings["sitemap_enabled"] == "false"` by emitting an empty `<urlset>` (or the root only) — this closes the Phase 1 carried-forward note that `sitemap_enabled` was seeded but unread. Keep the existing manual XML writing; escape slugs with `xml.EscapeText`.

- [ ] **Step 4: Run the tests and full gate, then commit**

```bash
git add internal/web/seo.go internal/web/seo_test.go
git commit -m "feat: gate posts in the sitemap and honour sitemap_enabled"
```

---

### Task 7: Admin post CRUD UI

**Files:**
- Create: `internal/web/admin_posts.go`
- Create: `internal/web/admin_posts_test.go`
- Create: `internal/web/templates/admin_posts.html`
- Create: `internal/web/templates/admin_post_form.html`
- Modify: `internal/web/render.go` (page fields if needed)
- Modify: `internal/web/templates/admin_layout.html` (nav link)
- Modify: `internal/web/web.go`

**Interfaces:**
- Consumes: `PostRepo`, `parseCSVTags(string) []db.Tag` (or `[]string` slugs), `formInt`, `renderAdmin`, CSRF middleware.
- Produces: `adminPostsGetHandler`, `adminPostNewHandler`, `adminPostEditHandler`, `adminPostCreateHandler`, `adminPostUpdateHandler`, `adminPostDeleteHandler`, `adminPostPublishHandler`, `adminPostUnpublishHandler`.

- [ ] **Step 1: Write the failing tests**

Mirror `admin_projects_test.go`: list renders posts; create with tags persists and `Reload`s; edit renders current values; update changes fields; delete 303s and removes; publish flips status and sets `published_at`; a hidden/non-existent id 404s; the forms carry a server-rendered `csrf_token`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminPost -v`
Expected: FAIL — no routes/handlers.

- [ ] **Step 3: Implement**

Handlers parse the form (`title`, `slug` defaulted from a slugified title, `summary`, `body_md`, `hero_image_url`, `hero_image_alt`, `seo_*`, `tags` as a comma-separated field), call the repo, `d.Content.Reload()`, and 303 to `/admin/posts?saved=1`. Register routes inside the existing `r.Route("/admin", ...)`:

```
GET  /admin/posts                 adminPostsGetHandler
GET  /admin/posts/new             adminPostNewHandler
POST /admin/posts                 adminPostCreateHandler
GET  /admin/posts/{id}            adminPostEditHandler
POST|PUT /admin/posts/{id}        adminPostUpdateHandler
POST|DELETE /admin/posts/{id}/delete   adminPostDeleteHandler
POST|PUT /admin/posts/{id}/publish     adminPostPublishHandler
POST|PUT /admin/posts/{id}/unpublish   adminPostUnpublishHandler
```

Every write form carries `<input type="hidden" name="csrf_token" value="{{.CSRF}}">`. Use a triple-quoted `<textarea name="body_md">{{.Post.BodyMD}}</textarea>` (html/template escapes it correctly for a textarea).

- [ ] **Step 4: Run the tests and full gate, then commit**

```bash
git add internal/web/admin_posts.go internal/web/admin_posts_test.go internal/web/templates/admin_posts.html internal/web/templates/admin_post_form.html internal/web/templates/admin_layout.html internal/web/render.go internal/web/web.go
git commit -m "feat: add admin post CRUD and publish controls"
```

---

### Task 8: Posts REST API (public and admin)

**Files:**
- Create: `internal/web/api_posts.go`
- Create: `internal/web/api_admin_posts.go`
- Create: `internal/web/api_posts_test.go`
- Modify: `internal/web/web.go`
- Modify: `internal/db/backup.go` (posts/tags in export/import)
- Modify: `internal/db/backup_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1 and 7, `writeJSON`, `writeAPIError`, `decodeJSON`, `apiID`, `apiAuthMiddleware`.
- Produces: `apiPostsHandler`, `apiPostHandler`, and an admin CRUD handler set mirroring `api_admin_content.go` (the generic `crudRepo[T]` does not fit posts because of tags, publish, and draft filtering — write explicit handlers, but reuse `writeJSON`/`decodeJSON`/`apiID`).

- [ ] **Step 1: Write the failing tests**

- `TestAPIPostsListsPublishedOnly`: drafts absent; `?tag=` filters; `?page=2` paginates; response includes a `total`/`page` envelope.
- `TestAPIPostBySlug`: a draft slug is 404; a published slug returns the post with `body_md` and rendered? — the API returns `body_md` raw plus `html` rendered, per the spec's "agent-first" intent; assert both.
- `TestAPIPostsDisabled`: with the switch off, `/api/v1/posts` and `/api/v1/posts/{slug}` return 404.
- `TestAPIAdminPostsCRUD`: create/update/delete/publish/unpublish with a session cookie; 401 without credentials.
- `TestBackupIncludesPosts`: export contains posts/tags; import restores them; a snapshot with `Posts` nil is rejected (extend the Task 5 pattern).

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAPIPosts|TestAPIAdminPosts' -v`
Expected: FAIL — routes absent.

- [ ] **Step 3: Implement the public API**

`apiPostsHandler`: if `!postsEnabled` → 404. Parse `page`, `tag`; return `{"posts":[...],"page":N,"per_page":10,"total":N}`. For each post set `html = markdown.Render(body_md)` as a `template.HTML`-backed string (the JSON encoder will emit the HTML string; no escaping concerns because it is served as JSON data, not HTML). `apiPostHandler`: `PublishedBySlug`; miss → 404.

Register under the existing `/api/v1` group (before the admin subgroup): `api.Get("/posts", ...)`, `api.Get("/posts/{slug}", ...)`.

- [ ] **Step 4: Implement the admin API**

Under `apiAdmin`: `GET|POST /posts`, `GET|PUT|DELETE /posts/{id}`, `POST /posts/{id}/publish`, `POST /posts/{id}/unpublish`. Create/update accept a JSON post body with a `tags` array of strings. Each write calls `d.Content.Reload()`. Reject `status` values other than `draft`/`published` with 400.

- [ ] **Step 5: Extend export/import**

Add `Posts []Post` and `Tags []Tag` to `db.Snapshot`. `Export` loads them; `Import` requires `Posts != nil` and `Tags != nil` (same rule as the content sections), deletes `post_tag`/`post`/`tag` in that order, and re-inserts preserving ids and tags. `post_tag` foreign keys cascade, so delete `post` before `tag`. Update the existing tests and add a round-trip.

- [ ] **Step 6: Run the tests and full gate, then commit**

```bash
git add internal/web/api_posts.go internal/web/api_admin_posts.go internal/web/api_posts_test.go internal/web/web.go internal/db/backup.go internal/db/backup_test.go
git commit -m "feat: add the posts REST API and include posts in backups"
```

---

### Task 9: Documentation, end-to-end verification, and definition of done

**Files:**
- Modify: `README.md`, `AGENTS.md`, `CHANGELOG.md`

- [ ] **Step 1: README** — add a `## Posts` section (authoring via `/admin/posts`, drafts vs published, tags, RSS at `/feed.xml`, the `posts_enabled` switch and what it 404s), add the posts routes to the API section, and add the new files to the architecture tree.

- [ ] **Step 2: AGENTS.md** — add the posts/feed routes, the `internal/db/repo_post.go`, `internal/web/posts.go`, `internal/web/feed.go`, `internal/web/admin_posts.go` and API files to the tree, and record the invariants: public post reads filter `status = 'published'`; every admin write calls `Reload`; the `posts_enabled` switch 404s every public post surface.

- [ ] **Step 3: CHANGELOG.md** — add the posts feature under `[Unreleased] → Added`.

- [ ] **Step 4: Manual E2E** — run against `/tmp/jf-phase4`, create a post through `/admin/posts`, publish it, verify `/posts`, `/posts/{slug}`, `/tags/{slug}`, `/feed.xml`, `/api/v1/posts`, and the sitemap; then set `posts_enabled=false` and confirm each returns 404 while the post remains in the DB.

- [ ] **Step 5: Final gate and commit**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`

```bash
git add README.md AGENTS.md CHANGELOG.md
git commit -m "docs: document the posts system"
```

---

## Phase 4a Definition of Done

- `post`, `tag`, `post_tag` exist via append-only migration `0006`; repositories cover create/update/delete/publish/list/by-tag/by-slug.
- Markdown renders GFM with Chroma class-based highlighting, raw HTML escaped, and no inline styles.
- `/posts` (paginated), `/posts/{slug}`, `/tags/{slug}` render; drafts are 404 everywhere; `?tag=` filters.
- `/feed.xml` is valid RSS 2.0 with RFC 1123 `pubDate`.
- `posts_enabled=false` 404s every public post surface, hides the nav link, and omits posts from the sitemap, without deleting data.
- Admin CRUD with tags, hero image, publish/unpublish, and CSRF-protected forms.
- Public and admin posts REST endpoints with drafts hidden and writes authenticated.
- Export/import includes posts and tags.
- Build, test, vet, gofmt all clean; docs updated.

## Out of scope for Phase 4a

- `page_seo` table, global SEO settings editing, per-post SEO override precedence, `BlogPosting` structured data, canonical/OG resolution overrides (Phase 4b).
- MCP tools (Phase 5).
- Theme library and Chroma themes derived per active theme (Phase 6) — 4a uses one token-mapped stylesheet.
- Visitor stats and page-view recording (Phase 7).
- Docker, CI, `/docs` reference page (Phase 8).
