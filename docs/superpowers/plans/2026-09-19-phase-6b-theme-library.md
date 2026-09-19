# Phase 6b — Theme Library, import_theme and Theme MCP Tools

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Ship the seeded theme library, an `import_theme` palette importer, and the theme MCP tools, so the site starts with a set of real design-system themes and an agent can list, create, update, delete, activate and import themes.

**Architecture:** `internal/theme` gains a `Seeds()` library (Catppuccin via `github.com/catppuccin/go` plus six hand-authored palettes) and a `Palette` type that maps a small colour set onto the token vocabulary. `internal/db` gains `SeedThemes` (insert-if-missing) called once from `cmd/johansenfoo/main.go`. `internal/mcp` gains theme tools that reuse `db.ThemeRepo` and `theme.Validate` (already imported by the repo).

**Tech Stack:** Go 1.25, `github.com/catppuccin/go`, `modernc.org/sqlite`, `github.com/mark3labs/mcp-go`.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Theming seed library; MCP theme tools)

## Global Constraints

- `CGO_ENABLED=0`; module `github.com/mojoaar/johansenfoo`, Go 1.25.5. No comments unless essential.
- Migrations append-only; this phase adds none (seeds are inserted at startup, not by migration).
- Every seeded/imported theme must pass `theme.Validate`; UI guard rails (base/active protected) already live in `ThemeRepo`.
- MCP tool errors use `mcp.NewToolResultError`; writes call `Reload`.
- Every task ends with build/test/vet/gofmt clean, then a commit.
- Docs are part of the definition of done.

## Scope

In scope: `theme.Seeds()` (Catppuccin Latte/Frappé/Macchiato/Mocha + Nord, Rosé Pine, Tokyo Night, Gruvbox, Everforest, Solarized), startup seeding, `import_theme`, and the seven theme MCP tools.

Out of scope: per-theme Chroma stylesheet generation; layout tokens.

## Review Focus

1. **Seeding is idempotent and non-destructive** — it inserts a theme only when its slug is absent and never overwrites an edited theme; every seeded theme validates.
2. **import_theme validates** — an imported palette with an unknown token or a breaking value is rejected, not persisted; a duplicate slug is a tool error.
3. **MCP theme tool guard rails** — `delete_theme` refuses the base and active themes; `set_active_theme` refuses an unknown slug.
4. **Reload** — every theme write tool reloads so `/` picks up a newly active theme.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/theme/palette.go` | `Palette` type and token mapping |
| `internal/theme/seeds.go` | `Seeds()` library |
| `internal/theme/seeds_test.go` | Library validation tests |
| `internal/db/seed_themes.go` | `SeedThemes(*sql.DB, []theme.Theme) (int, error)` |
| `internal/db/seed_themes_test.go` | Idempotency tests |
| `cmd/johansenfoo/main.go` | Call `SeedThemes` after `Migrate` |
| `internal/mcp/tools_themes.go` | Theme tools |
| `internal/mcp/tools_test.go` | Theme tool tests |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Docs |

---

### Task 1: Palette mapping and the seed library

**Files:** create `internal/theme/palette.go`, `internal/theme/seeds.go`, `internal/theme/seeds_test.go`.

**Interfaces:**
- `type Palette struct { Bg, Bg2, Bg3, Border, Text, Muted, Accent, Accent2, Green string }`
- `func (p Palette) Tokens() map[string]string` — fills the 11 required tokens (`--accent-glow` = accent, `--shadow` fixed).
- `func Seeds() []Theme` — 10 themes: `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-mocha`, `nord`, `rose-pine`, `tokyo-night`, `gruvbox`, `everforest`, `solarized`.

- [ ] **Step 1: Failing tests** — `Seeds()` has exactly 10 entries; slugs unique; every theme passes `Validate`; the mocha theme's dark `--accent` equals the catppuccin Mauve hex.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** `Palette.Tokens`, a `newTheme(slug,name,desc string, light, dark Palette) Theme`, catppuccin flavours via `catppuccingo.Mocha` etc. (light set = `catppuccingo.Latte`, dark set = the flavour; for Latte, dark = Mocha), and six hand-authored palettes.
- [ ] **Step 4: Pass + gate + commit** `feat: add the seeded theme library`.

---

### Task 2: Startup seeding

**Files:** create `internal/db/seed_themes.go`, `internal/db/seed_themes_test.go`; modify `cmd/johansenfoo/main.go`.

**Interfaces:** `db.SeedThemes(d *sql.DB, themes []theme.Theme) (int, error)` — for each theme, insert only if `GetBySlug` returns `ErrThemeNotFound`; returns the count inserted.

- [ ] **Step 1: Failing tests** — seeding an empty DB inserts all; a second call inserts 0; an edited existing theme is not overwritten (change the name, re-seed, name unchanged).
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement**, call it from `main.go` after `db.Migrate` (log the count only on error or a debug line).
- [ ] **Step 4: Pass + gate + commit** `feat: seed the theme library at startup`.

---

### Task 3: `import_theme` and the MCP theme tools

**Files:** create `internal/mcp/tools_themes.go`; modify `internal/mcp/tools_test.go`, `internal/mcp/server.go`.

**Interfaces (tools → backend methods):**
- `list_themes` → `ListThemes()` (returns themes with tokens)
- `get_theme(slug)` → `GetTheme(slug)`
- `create_theme(slug, name, description, tokens_base, tokens_light, tokens_dark)` → `CreateTheme`
- `update_theme(slug, ... optional token maps ...)` → `UpdateTheme` (merges)
- `delete_theme(slug)` → `DeleteTheme` (base/active → tool error)
- `set_active_theme(slug)` → `SetActiveTheme`
- `import_theme(name, slug?, flavour?, tokens_base?, tokens_light?, tokens_dark?)` → `ImportTheme`; when `flavour` is one of the catppuccin variants, take that flavour's palette; otherwise build from the supplied token maps; validate and create.

- [ ] **Step 1: Failing tests** — `list_themes` includes `johansen`; `create_theme` with a valid map is listed; `create_theme` with a `</style>` value is a tool error; `update_theme` merges; `delete_theme` of `johansen` is a tool error; `set_active_theme` of a missing slug is a tool error and of a created slug reloads; `import_theme` with `flavour:"mocha"` creates a valid theme.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** and register in `registerThemeTools` (called from `NewServer`).
- [ ] **Step 4: Pass + gate + commit** `feat: add MCP theme tools and import_theme`.

---

### Task 4: Documentation, end-to-end verification, definition of done

- [ ] **Step 1: Docs** — README theme section (seed library list, `import_theme`), AGENTS tree/tools, CHANGELOG.
- [ ] **Step 2: Manual E2E** — run against `/tmp/jf-6b`; `tools/list` includes the theme tools; `list_themes` shows the seeded library; `set_active_theme nord` changes `data-theme` on `/`; `import_theme` with a catppuccin flavour creates a theme; `delete_theme johansen` is a tool error.
- [ ] **Step 3: Final gate.**
- [ ] **Step 4: Commit** `docs: document the theme library and import_theme`.

## Phase 6b Definition of Done

- 10 seeded themes at startup, idempotent and non-destructive, all validated.
- `import_theme` ingests a Catppuccin flavour or a token map and validates.
- The seven theme MCP tools work with guard rails and Reload.
- Gates clean; docs updated.
