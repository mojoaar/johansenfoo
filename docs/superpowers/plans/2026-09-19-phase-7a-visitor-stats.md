# Phase 7a — Visitor Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Record privacy-preserving page views and surface them — a page_view table, a middleware that records public HTML GETs with a daily-rotating salted IP hash, a daily retention job, admin dashboard visitor cards, `GET|DELETE /api/v1/admin/stats/visitors`, and the `get_visitor_stats` / `clear_visitor_stats` MCP tools.

**Architecture:** Migration `0008` adds `page_view`. `internal/db/repo_views.go` owns recording, aggregation and pruning. `internal/web` holds a `viewRecorder` (holds today's salt in memory, refreshing on date change) and the `pageViewMiddleware`, applied only to public routes and only recording `text/html` 2xx GETs. `stats_enabled=false` disables recording; a daily job prunes rows older than `stats_retention_days`. No cookies, no raw IPs — only a salted SHA-256 that rotates daily, so uniqueness is per-day and 7d/30d figures are sums of daily uniques.

**Tech Stack:** Go 1.25, `modernc.org/sqlite`, `net/http`, `crypto/rand`, `crypto/sha256`, `html/template`.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Visitor Stats & Privacy; admin stats endpoints; MCP stats tools)

## Global Constraints

- `CGO_ENABLED=0`; module `github.com/mojoaar/johansenfoo`, Go 1.25.5. No comments unless essential.
- Timestamps RFC 3339 UTC via `strftime('%Y-%m-%dT%H:%M:%SZ','now')`; all SQL parameterized.
- Migrations append-only: `0001`-`0007` frozen; this phase adds `0008`.
- Recording excludes `/admin`, `/static`, `/api`, `/mcp`, `/metrics` and only stores GET 2xx HTML requests on public routes.
- No cookies are set by stats; the IP is never stored raw; the salt rotates daily.
- Every successful admin write calls `d.Content.Reload()` (stats are not content, so dashboard/retention do not).
- Every task ends with build/test/vet/gofmt clean, then a commit.
- Docs are part of the definition of done.

## Scope

In scope: `page_view`, recording, retention, visitor aggregation, dashboard visitor cards, REST visitors endpoint, MCP `get_visitor_stats` / `clear_visitor_stats`.

Out of scope for 7a (7b): Prometheus `/metrics`, `internal/sysinfo` runtime/container stats, runtime dashboard cards, `stats/system` REST, `get_system_stats`.

## Review Focus

1. **Privacy** — no raw IP is ever persisted; the same IP on different days hashes differently; no cookie is set by stats.
2. **Exclusions** — admin, static, API, MCP and metrics traffic is never recorded; JSON/`/me` and non-GET are not recorded.
3. **Kill switch** — `stats_enabled=false` records nothing and the dashboard shows a disabled state without erroring.
4. **Retention** — rows older than `stats_retention_days` are pruned; the job is testable via a seam.
5. **Daily uniques** — uniqueness is per day, so 7d/30d are sums of daily uniques (documented and tested).

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/db/migrations/0008_page_view.sql` | `page_view` table + created_at index |
| `internal/db/repo_views.go` | `PageViewRepo`: Record, counts, daily uniques, top pages/referrers, recent, Clear, Prune |
| `internal/db/repo_views_test.go` | Repository tests |
| `internal/web/stats.go` | salt/`viewRecorder`, `pageViewMiddleware`, retention job, aggregations |
| `internal/web/stats_test.go` | Middleware, privacy, exclusion, kill-switch tests |
| `internal/web/admin.go`, `templates/admin_dashboard.html` | Visitor cards |
| `internal/web/api_admin_stats.go` | `GET|DELETE /api/v1/admin/stats/visitors` |
| `internal/web/api_admin_stats_test.go` | REST tests |
| `internal/mcp/tools_stats.go` | `get_visitor_stats`, `clear_visitor_stats` |
| `internal/web/web.go`, `render.go` | Route + page fields |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Docs |

---

### Task 1: `page_view` schema and repository

**Files:** create `internal/db/migrations/0008_page_view.sql`, `internal/db/repo_views.go`, `internal/db/repo_views_test.go`; modify `internal/db/db_test.go`.

**Interfaces:**
- `db.PageView{Path, Referrer, UserAgent, IPHash string; CreatedAt time.Time}`
- `db.NewPageViewRepo(*sql.DB)`
- `Record(v *PageView) error`
- `CountSince(since string) (int, error)`; `CountBetween(from, to string) (int, error)`
- `DailyUniqueCount(day string) (int, error)` (distinct ip_hash for a `YYYY-MM-DD` day)
- `TopPaths(since string, limit int) ([]PathCount, error)`; `TopReferrers(since string, limit int) ([]ReferrerCount, error)`
- `Recent(limit int) ([]PageView, error)`
- `Prune(before string) (int64, error)`; `Clear() error`

- [ ] **Step 1: Failing tests** — record + count; `DailyUniqueCount` counts distinct hashes and ignores other days; `TopPaths`/`TopReferrers` order by count desc; `Prune` removes only older rows; `Clear` empties.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Migration**
```sql
CREATE TABLE IF NOT EXISTS page_view (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    path       TEXT NOT NULL,
    referrer   TEXT,
    user_agent TEXT,
    ip_hash    TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_page_view_created_at ON page_view (created_at);
```
- [ ] **Step 4: Implement the repo** with `strftime('%Y-%m-%dT%H:%M:%SZ','now')` for inserts and day ranges computed as `substr(created_at, 1, 10) = ?`.
- [ ] **Step 5: Pass + gate + commit** `feat: add the page_view table and repository`.

---

### Task 2: Salted IP hashing, the recording middleware and retention

**Files:** create `internal/web/stats.go`, `internal/web/stats_test.go`; modify `internal/web/web.go`, `internal/db/repo_settings.go` if needed.

**Interfaces:**
- `type viewRecorder struct { db *sql.DB; mu sync.Mutex; salt string; day string }`
- `func newViewRecorder(*sql.DB) *viewRecorder`; `(*viewRecorder).hash(ip string) (string, error)` — ensures today's salt (random 16 bytes, stored in `settings` under `stats_salt`/`stats_salt_date`), returns `sha256(salt + ":" + ip)` hex.
- `func pageViewMiddleware(d Deps) func(http.Handler) http.Handler` — after the handler, for `GET` 2xx with `Content-Type: text/html`, if `stats_enabled != "false"`, record path, referrer, UA, hashed `clientIP(r)`.
- `func startStatsPruner(d *sql.DB, interval time.Duration) (stop func())` (mirrors the session pruner) and `pruneViews(d *sql.DB) (int64, error)` reading `stats_retention_days`.

- [ ] **Step 1: Failing tests** — `hash` is stable within a day and differs when the day changes (inject the day/salt); middleware records a 200 HTML GET; does **not** record a JSON response, a 404, or a POST; a `/admin` request is not recorded (mount order); `stats_enabled=false` records nothing; `pruneViews` removes old rows per setting.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement**, apply `pageViewMiddleware` to the public route registrations only (not the admin/API groups), and start `startStatsPruner` in `web.New` alongside the session pruner.
- [ ] **Step 4: Pass + gate + commit** `feat: record privacy-preserving page views`.

---

### Task 3: Dashboard visitor cards, REST and MCP

**Files:** create `internal/web/api_admin_stats.go`, `internal/web/api_admin_stats_test.go`, `internal/mcp/tools_stats.go`; modify `internal/web/admin.go`, `internal/web/templates/admin_dashboard.html`, `internal/web/render.go`, `internal/web/web.go`, `internal/mcp/tools_test.go`, `internal/mcp/server.go`.

**Interfaces:**
- `page` gains `StatsEnabled bool`, `ViewsToday, Views7d, Views30d, DailyUniques int`, `TopPaths []db.PathCount`, `TopReferrers []db.ReferrerCount`, `RecentHits []db.PageView`.
- REST `GET /api/v1/admin/stats/visitors?period=7d` → `{period, views, daily_uniques, top_paths, top_referrers, recent}`; `DELETE /api/v1/admin/stats/visitors` clears.
- MCP `get_visitor_stats(period?)` and `clear_visitor_stats()`.

- [ ] **Step 1: Failing tests** — dashboard renders the counts and, with `stats_enabled=false`, a disabled state; the REST endpoint returns the envelope for `period=7d` and 401 unauthenticated; DELETE clears; MCP `get_visitor_stats` returns counts and `clear_visitor_stats` empties.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** the dashboard data, template cards, REST handlers (register under `apiAdmin`) and MCP tools (`registerStatTools`).
- [ ] **Step 4: Pass + gate + commit** `feat: add visitor stats to the dashboard, REST and MCP`.

---

### Task 4: Documentation, end-to-end verification, definition of done

- [ ] **Step 1: Docs** — README privacy/stats section, AGENTS architecture/routes/tools and privacy invariant, CHANGELOG.
- [ ] **Step 2: Manual E2E** — run against `/tmp/jf-7a`; hit `/` and `/posts` a few times and `/me` once; confirm `page_view` rows only for the HTML pages; confirm `ip_hash` is not the raw IP; set `stats_enabled=false` and confirm no new rows; call `get_visitor_stats` over MCP.
- [ ] **Step 3: Final gate.**
- [ ] **Step 4: Commit** `docs: document visitor stats`.

## Phase 7a Definition of Done

- `page_view` via append-only `0008`; recording covers public HTML GETs only, with a daily-rotating salted hash and no cookies.
- `stats_enabled` disables recording; retention prunes per `stats_retention_days`.
- Dashboard visitor cards, `GET|DELETE /api/v1/admin/stats/visitors`, and the two MCP tools.
- Gates clean; docs updated.

## Out of scope for 7a

- Prometheus `/metrics`, `internal/sysinfo`, runtime dashboard cards, `stats/system`, `get_system_stats` (7b).
