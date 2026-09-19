# Phase 6a — Theme Engine and Editor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make themes a first-class, editable, validated resource: widen the token vocabulary, close the carried-forward CSS-injection gate, give the theme repository full CRUD with guard rails, expose themes over REST, and add an admin theme list with a token editor and "set active".

**Architecture:** `internal/theme` owns the vocabulary, validation and CSS emission. `internal/db` gains theme CRUD (the existing repo only reads by slug). `internal/web` gains an admin editor and REST handlers. The active theme stays a `settings` row (`active_theme`); changing it reloads the content snapshot. `theme.Validate` becomes the single write-path gate: unknown tokens and any HTML/CSS-breaking characters are rejected before a row is written or CSS is emitted.

**Tech Stack:** Go 1.25, `modernc.org/sqlite`, `net/http`, `encoding/json`, `html/template`.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Theming; Authenticated Surface themes endpoints)

## Global Constraints

- `CGO_ENABLED=0`; module `github.com/mojoaar/johansenfoo`, Go 1.25.5. No comments unless essential.
- All SQL parameterized; timestamps RFC 3339 UTC via `strftime('%Y-%m-%dT%H:%M:%SZ','now')`.
- Migrations append-only: `0001`-`0007` frozen; this phase adds none (tokens live in JSON columns).
- `<html>` keeps `data-theme="<slug>"` and `data-mode="light|dark"` as separate axes.
- **Carried-forward gate:** no theme write path may emit CSS or a token that breaks out of `<style>`; `Validate` rejects it and `CSS` only ever receives validated values.
- Guard rails: the base `johansen` theme cannot be deleted; the active theme cannot be deleted.
- Every successful write calls `d.Content.Reload()`.
- Every task ends with build/test/vet/gofmt clean, then a commit.
- Docs are part of the definition of done.

## Scope

In scope: widened vocabulary, hardened `Validate`, `CSS` safety, theme repository CRUD, guard rails, `GET|POST|GET /{id}|PUT /{id}|DELETE /{id} /api/v1/admin/themes`, admin list/editor/set-active/delete.

Out of scope for 6a (6b): the seeded theme library (`catppuccin/go`, Nord, Rosé Pine, Tokyo Night, Gruvbox, Everforest, Solarized), `import_theme`, and the theme MCP tools.

## Review Focus

1. **CSS/HTML injection.** A token value containing `</style>`, `<`, `>` or `;` must be rejected on every write path and must never reach `CSS` output.
2. **Unknown tokens.** Any token name outside the vocabulary is rejected; adding values never needs a migration.
3. **Partial updates merge.** `update_theme` merges supplied token maps over the stored ones; a token removed from the payload is not silently dropped unless explicitly set empty.
4. **Guard rails.** Deleting `johansen` or the active theme is refused; deleting another theme leaves the active one untouched.
5. **Active theme correctness.** Setting the active theme to a missing slug is refused; a successful set is visible on the next public request (Reload).

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/theme/theme.go` | Widened vocabulary, hardened `Validate`, `Resolve` |
| `internal/theme/css.go` | `CSS` (unchanged shape, validated input) |
| `internal/theme/theme_test.go` | Vocabulary, validation, injection tests |
| `internal/db/repo_theme.go` | `List`, `Create`, `Update`, `Delete`, `SetActive`, `ActiveSlug` |
| `internal/db/repo_theme_test.go` | Repository + guard-rail tests |
| `internal/web/admin_themes.go` | Admin handlers |
| `internal/web/templates/admin_themes.html`, `admin_theme_form.html` | Admin UI |
| `internal/web/api_admin_themes.go` | REST handlers |
| `internal/web/api_admin_themes_test.go` | REST tests |
| `internal/web/web.go` | Routes |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Docs |

---

### Task 1: Widen the vocabulary and harden validation

**Files:** modify `internal/theme/theme.go`, `internal/theme/theme_test.go`.

**Interfaces:** `Validate(t Theme) error` keeps its signature; `requiredTokens` unchanged (the 11 base tokens every theme still needs); `optionalTokens` grows to the spec vocabulary. `known()` covers both. Rejection set for values becomes `{ } ; < >`.

New optional tokens: colour pairs `--card`, `--card-foreground`, `--popover`, `--popover-foreground`, `--primary`, `--primary-foreground`, `--secondary`, `--secondary-foreground`, `--muted`, `--muted-foreground`, `--accent-foreground`, `--destructive`, `--destructive-foreground`, `--input`, `--ring`; typography `--font-sans`, `--font-mono`, `--font-size-base`, `--line-height`, `--letter-spacing`, `--font-weight-normal`, `--font-weight-bold`; shape/depth `--radius`, `--border-width`, `--shadow`, `--shadow-sm`, `--shadow-lg`; code `--code-bg`, `--code-text`, `--code-keyword`, `--code-string`, `--code-comment`, `--code-function`, `--code-number`, `--code-operator`.

- [ ] **Step 1: Failing tests** — `Validate` accepts every new token in base/light/dark; rejects `unknown-token`; rejects values containing `<`, `>`, `;`, `{`, `}` (table); the base theme still validates unchanged.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** the widened `optionalTokens` and the rejection set.
- [ ] **Step 4: Pass + gate + commit** `feat: widen the theme token vocabulary and harden validation`.

---

### Task 2: Theme repository CRUD and guard rails

**Files:** modify `internal/db/repo_theme.go`, `internal/db/repo_theme_test.go`; possibly `internal/db/models.go`.

**Interfaces:**
- `(*ThemeRepo).List() ([]Theme, error)` ordered by `sort, id`.
- `(*ThemeRepo).ActiveSlug() (string, error)` (settings lookup, default `johansen`).
- `(*ThemeRepo).Create(*Theme) (int64, error)`, `Update(*Theme) error` (merges token maps: for each section, new keys overwrite, absent keys preserved), `Delete(id int64) error` (refuses slug `johansen` and the active slug with `ErrThemeProtected`), `SetActive(slug string) error` (verifies the slug exists).
- `db.ErrThemeNotFound`, `db.ErrThemeProtected`.

- [ ] **Step 1: Failing tests** — create/list/get round-trip; update merges partial tokens and leaves others; delete refuses `johansen` and the active theme, allows a third theme; `SetActive` to a missing slug errors and to an existing one succeeds; `ActiveSlug` defaults to `johansen`.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement**, running `theme.Validate` before every create/update write.
- [ ] **Step 4: Pass + gate + commit** `feat: add theme CRUD with guard rails`.

---

### Task 3: Admin theme list, editor and set-active

**Files:** create `internal/web/admin_themes.go`, `internal/web/templates/admin_themes.html`, `internal/web/templates/admin_theme_form.html`, `internal/web/admin_themes_test.go`; modify `internal/web/web.go`, `internal/web/templates/admin_layout.html`.

**Interfaces:** `adminThemesGetHandler`, `adminThemeNewHandler`, `adminThemeEditHandler`, `adminThemeCreateHandler`, `adminThemeUpdateHandler`, `adminThemeDeleteHandler`, `adminThemeActivateHandler`.

- [ ] **Step 1: Failing tests** — list renders theme names and the active marker; create persists and reloads; edit renders current token JSON; update merges; delete refuses the base theme (400) and removes a new one; activate sets `active_theme` and the public page uses the new slug; forms carry `csrf_token`.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** handlers + templates (list with Set active / Edit / Delete; form with name, slug, description, and three JSON textareas for base/light/dark), nav link `/admin/themes`, routes:
```
GET  /admin/themes                 adminThemesGetHandler
GET  /admin/themes/new             adminThemeNewHandler
POST /admin/themes                 adminThemeCreateHandler
GET  /admin/themes/{id}            adminThemeEditHandler
POST|PUT /admin/themes/{id}        adminThemeUpdateHandler
POST|DELETE /admin/themes/{id}/delete   adminThemeDeleteHandler
POST|PUT /admin/themes/{id}/activate    adminThemeActivateHandler
```
- [ ] **Step 4: Pass + gate + commit** `feat: add the admin theme editor`.

---

### Task 4: Theme REST endpoints

**Files:** create `internal/web/api_admin_themes.go`, `internal/web/api_admin_themes_test.go`; modify `internal/web/web.go`.

**Interfaces:** `GET|POST /api/v1/admin/themes`; `GET|PUT|DELETE /api/v1/admin/themes/{id}`; `POST /api/v1/admin/themes/{id}/activate`. Themes are returned as their full token maps; create/update validate before writing; delete returns 409 for a protected theme; activate returns 200 and reloads.

- [ ] **Step 1: Failing tests** — list; create (201) with validation failure on a bad token (400); get; update merges; delete a new theme (204) and a protected one (409); activate; unauthenticated 401; `PUT /api/v1/admin/settings/theme` still works.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement + register** under the existing `apiAdmin` group.
- [ ] **Step 4: Pass + gate + commit** `feat: add the themes REST API`.

---

### Task 5: Documentation, end-to-end verification, definition of done

- [ ] **Step 1: Docs** — README theme section (vocabulary, admin editor, REST), AGENTS architecture/routes and the "validate + no `<`/`>`" invariant, CHANGELOG.
- [ ] **Step 2: Manual E2E** — run against `/tmp/jf-6a`; create a theme via the admin API that overrides `--accent`; activate it; confirm the public page's inline `<style>` carries the new value and `data-theme` is the new slug; attempt to delete `johansen` → 409; attempt to inject `</style><script>` as a token → 400.
- [ ] **Step 3: Final gate** build/test/vet/gofmt.
- [ ] **Step 4: Commit** `docs: document the theme engine`.

## Phase 6a Definition of Done

- The vocabulary covers the spec's colour pairs, typography, shape/depth and code tokens; unknown tokens and HTML-breaking values are rejected.
- Theme CRUD with the two guard rails; partial updates merge.
- Admin theme editor and `GET|POST|GET|PUT|DELETE /api/v1/admin/themes` plus activate.
- Active theme changes are visible on the next public request.
- Gates clean; docs updated.

## Out of scope for 6a

- Seeded theme library and `import_theme` (6b).
- Theme MCP tools (6b).
- Per-theme Chroma stylesheet generation — `code.css` remains token-mapped.
- Any layout-affecting token (the spec forbids it).
