# Phase 1 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up the `johansenfoo` Go service so it serves the existing public landing page, rendered from SQLite, at pixel parity with the current static site.

**Architecture:** A single Go binary. Content lives in SQLite (CGO-free `modernc.org/sqlite`), accessed through repositories. `html/template` renders the page server-side from that data on every request. The active theme is emitted as an inline `<style>` block in `<head>`, so the stylesheet keeps using CSS custom properties with no extra request. Icons are inline SVG produced by Go from vendored sources — no Lucide runtime, no Font Awesome.

**Tech Stack:** Go 1.25, `github.com/go-chi/chi/v5`, `modernc.org/sqlite`, `html/template`, `github.com/yuin/goldmark`.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md`

## Global Constraints

- Module path: `github.com/mojoaar/johansenfoo`. Go directive: `go 1.25.5`.
- No CGO. Build with `CGO_ENABLED=0`.
- No comments in Go code unless essential.
- No third-party origin on the critical path: no Google Fonts, no Font Awesome, no Lucide CDN. JetBrains Mono is self-hosted; icons are inline SVG.
- All timestamps stored as RFC 3339 in UTC.
- No JavaScript framework. Vanilla inline JS only, exactly as the current site.
- Structured content is stored as markdown and rendered with goldmark.
- `data-theme` carries the **theme slug**; `data-mode` carries **light|dark**. These are separate axes.
- Colour palette must match the current site exactly: dark `--bg: #0f1117`, light `--bg: #f4f6fb`.
- Every task ends with `go test ./...` and `go vet ./...` passing, then a commit.

## Scope Note

The spec's theming section describes the **end state** of the token vocabulary (shadcn surface/foreground pairs, typography, shape, syntax highlighting). Phase 1 ships only the token set the ported `style.css` actually consumes, because Phase 1's deliverable is pixel parity. The vocabulary is widened in Phase 6, when multiple themes and `import_theme` make the wider set necessary.

Similarly, `internal/markdown` is introduced here in minimal form (GFM, no highlighting, no sanitizer) because the hero bio contains a link and must render identically. Chroma highlighting and sanitisation arrive in Phase 4.

## File Structure

| Path | Responsibility |
| --- | --- |
| `go.mod`, `go.sum` | Module definition |
| `.gitignore`, `LICENSE`, `VERSION` | Repo metadata |
| `cmd/johansenfoo/main.go` | Entry point: flags, wiring, graceful shutdown |
| `cmd/johansenfoo/main_test.go` | Boots the server, asserts `/health` |
| `internal/config/config.go` | JSON config with defaults |
| `internal/db/db.go` | Opens SQLite, sets pragmas |
| `internal/db/migrate.go` | Embedded migration runner |
| `internal/db/migrations/0001_schema.sql` | Tables |
| `internal/db/migrations/0002_seed.sql` | Seeded content from the current site |
| `internal/db/models.go` | Structs shared by repos and templates |
| `internal/db/repo_profile.go` | Profile + social links |
| `internal/db/repo_content.go` | Projects, experience, skills |
| `internal/db/repo_settings.go` | Key/value settings |
| `internal/db/repo_theme.go` | Theme rows |
| `internal/markdown/markdown.go` | goldmark GFM renderer |
| `internal/theme/theme.go` | Token vocabulary and validation |
| `internal/theme/css.go` | Emits the CSS custom-property block |
| `internal/theme/johansen.go` | The `johansen` base theme's token values |
| `internal/icons/icons.go` | Vendored SVG → inline SVG |
| `internal/icons/svg/*.svg` | Vendored icon sources |
| `internal/web/web.go` | chi router, middleware, route table |
| `internal/web/render.go` | Template loading, func map, base layout |
| `internal/web/landing.go` | `GET /` |
| `internal/web/me.go` | `GET /me` |
| `internal/web/seo.go` | Metadata resolution, robots.txt, sitemap.xml |
| `internal/web/static.go` | Embedded assets and `/static/*` |
| `internal/web/static/style.css` | Ported stylesheet |
| `internal/web/static/fonts/*` | Self-hosted JetBrains Mono |
| `internal/web/static/favicon*`, `avatar.png` | Ported assets |
| `internal/web/templates/base.html` | Document skeleton, head, SEO block |
| `internal/web/templates/landing.html` | Landing page body |

---

### Task 1: Module scaffold, config, and health endpoint

**Files:**
- Create: `go.mod`, `.gitignore`, `LICENSE`, `VERSION`
- Create: `internal/config/config.go`, `internal/config/config_test.go`
- Create: `internal/web/web.go`, `internal/web/health.go`, `internal/web/web_test.go`
- Create: `cmd/johansenfoo/main.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `config.Config{Port int, DBPath string, BaseURL string}`, `config.Load(dir string) (*Config, error)`, `web.New(web.Deps) http.Handler`, `web.Deps{DB *sql.DB, Cfg *config.Config, Version string, Started time.Time}`.

- [ ] **Step 1: Initialise the module**

```bash
cd /Users/mojoaar/Development/johansenfoo
go mod init github.com/mojoaar/johansenfoo
go get github.com/go-chi/chi/v5@v5.3.1
```

Set the Go directive to match `icloud-mailflow`:

```bash
go mod edit -go=1.25.5
```

- [ ] **Step 2: Write `.gitignore`, `LICENSE`, `VERSION`**

`.gitignore`:

```
/data/
*.db
*.db-shm
*.db-wal
/bin/
.DS_Store
```

`VERSION` — a single line, no trailing newline:

```
0.1.0
```

`LICENSE` — MIT, `Copyright (c) 2026 Morten Johansen`. Copy the licence text verbatim from `/Users/mojoaar/Development/johansen_landing/LICENSE`.

- [ ] **Step 3: Write the failing config test**

`internal/config/config_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DBPath != filepath.Join(dir, "johansenfoo.db") {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, filepath.Join(dir, "johansenfoo.db"))
	}
	if cfg.BaseURL != "https://johansen.foo" {
		t.Errorf("BaseURL = %q, want https://johansen.foo", cfg.BaseURL)
	}
}

func TestLoadOverridesFromFile(t *testing.T) {
	dir := t.TempDir()
	body := `{"port": 9000, "db_path": "/tmp/custom.db", "base_url": "http://localhost:9000"}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("DBPath = %q, want /tmp/custom.db", cfg.DBPath)
	}
	if cfg.BaseURL != "http://localhost:9000" {
		t.Errorf("BaseURL = %q, want http://localhost:9000", cfg.BaseURL)
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatal("Load succeeded on malformed config, want error")
	}
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `go test ./internal/config/ -v`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 5: Implement the config package**

`internal/config/config.go`:

```go
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Port    int    `json:"port"`
	DBPath  string `json:"db_path"`
	BaseURL string `json:"base_url"`
}

func Load(dir string) (*Config, error) {
	cfg := &Config{
		Port:    8080,
		DBPath:  filepath.Join(dir, "johansenfoo.db"),
		BaseURL: "https://johansen.foo",
	}

	path := filepath.Join(dir, "config.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var onDisk Config
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		return nil, err
	}
	if onDisk.Port != 0 {
		cfg.Port = onDisk.Port
	}
	if onDisk.DBPath != "" {
		cfg.DBPath = onDisk.DBPath
	}
	if onDisk.BaseURL != "" {
		cfg.BaseURL = onDisk.BaseURL
	}
	return cfg, nil
}
```

- [ ] **Step 6: Write the failing health test**

`internal/web/web_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return New(Deps{
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Version: "test",
		Started: time.Now(),
	})
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "ok\n" {
		t.Errorf("body = %q, want %q", got, "ok\n")
	}
}
```

`internal/web/web.go` and `health.go` do not exist yet, and `New`/`Deps` are undefined, so this will not compile — which is the failure we want.

- [ ] **Step 7: Run the test to verify it fails**

Run: `go test ./internal/web/ -v`
Expected: FAIL — `undefined: New` / `undefined: Deps`.

- [ ] **Step 8: Implement the router and health handler**

`internal/web/web.go`:

```go
package web

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mojoaar/johansenfoo/internal/config"
)

type Deps struct {
	DB      *sql.DB
	Cfg     *config.Config
	Version string
	Started time.Time
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(securityHeaders)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthHandler(d))

	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
```

`internal/web/health.go`:

```go
package web

import "net/http"

func healthHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	}
}
```

- [ ] **Step 9: Run the test to verify it passes**

Run: `go test ./internal/web/ -v`
Expected: PASS.

- [ ] **Step 10: Write the entry point**

`cmd/johansenfoo/main.go`:

```go
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/web"
)

var version = "0.1.0"

func main() {
	dataDir := flag.String("data", "./data", "data directory")
	flag.Parse()

	if err := run(*dataDir); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	cfg, err := config.Load(dataDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	handler := web.New(web.Deps{
		Cfg:     cfg,
		Version: version,
		Started: time.Now(),
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", srv.Addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 11: Verify the whole build, then smoke-test by hand**

```bash
go vet ./... && go test ./...
go build -o /tmp/johansenfoo ./cmd/johansenfoo
/tmp/johansenfoo -data=/tmp/jf-smoke &
sleep 1
curl -sS -i http://localhost:8080/health
kill %1
```

Expected: `HTTP/1.1 200 OK`, body `ok`.

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "feat: scaffold Go service with config and health endpoint"
```

---

### Task 2: SQLite connection and migration runner

**Files:**
- Create: `internal/db/db.go`, `internal/db/migrate.go`, `internal/db/db_test.go`
- Create: `internal/db/migrations/0001_schema.sql`

**Interfaces:**
- Consumes: nothing.
- Produces: `db.Open(path string) (*sql.DB, error)`, `db.Migrate(d *sql.DB) error`.

- [ ] **Step 1: Add the SQLite driver**

```bash
go get modernc.org/sqlite@v1.55.0
```

- [ ] **Step 2: Write the failing migration test**

`internal/db/db_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"
)

func TestMigrateIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	if err := Migrate(d); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if n == 0 {
		t.Error("schema_migrations is empty, want at least one row")
	}
}

func TestSchemaHasCoreTables(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, table := range []string{
		"profile", "social_link", "project", "experience",
		"skill", "settings", "theme", "schema_migrations",
	} {
		var name string
		err := d.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q missing: %v", table, err)
		}
	}
}
```

`internal/db/db.go` and `internal/db/migrate.go` do not exist yet, so this will not compile — which is the failure we want.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/db/ -v`
Expected: FAIL — `undefined: Open`.

- [ ] **Step 4: Implement `Open`**

`internal/db/db.go`:

```go
package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	pragmas := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA foreign_keys = ON`,
		`PRAGMA synchronous = NORMAL`,
	}
	for _, p := range pragmas {
		if _, err := d.Exec(p); err != nil {
			_ = d.Close()
			return nil, err
		}
	}

	if err := d.Ping(); err != nil {
		_ = d.Close()
		return nil, err
	}
	return d, nil
}
```

- [ ] **Step 5: Write the schema migration**

`internal/db/migrations/0001_schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS profile (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    name          TEXT NOT NULL,
    handle        TEXT NOT NULL,
    location      TEXT NOT NULL,
    dob           TEXT NOT NULL,
    tagline       TEXT NOT NULL,
    hero_bio      TEXT NOT NULL,
    about_para_1  TEXT NOT NULL,
    about_para_2  TEXT NOT NULL,
    avatar        TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS social_link (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    platform TEXT NOT NULL,
    url      TEXT NOT NULL,
    label    TEXT NOT NULL,
    sort     INTEGER NOT NULL DEFAULT 0,
    visible  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS project (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    url         TEXT,
    description TEXT NOT NULL,
    icon        TEXT NOT NULL,
    is_link     INTEGER NOT NULL DEFAULT 1,
    url_label   TEXT NOT NULL DEFAULT '',
    sort        INTEGER NOT NULL DEFAULT 0,
    visible     INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS experience (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    years   TEXT NOT NULL,
    role    TEXT NOT NULL,
    company TEXT NOT NULL,
    icon    TEXT NOT NULL,
    sort    INTEGER NOT NULL DEFAULT 0,
    visible INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS skill (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    name    TEXT NOT NULL,
    sort    INTEGER NOT NULL DEFAULT 0,
    visible INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS theme (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    tokens_base  TEXT NOT NULL DEFAULT '{}',
    tokens_light TEXT NOT NULL DEFAULT '{}',
    tokens_dark  TEXT NOT NULL DEFAULT '{}',
    sort         INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
```

Note: three token columns, not two. `tokens_base` holds mode-independent tokens (`--radius`, `--font-mono`, `--font-sans`); `tokens_light` and `tokens_dark` hold the overrides for each mode.

- [ ] **Step 6: Implement the migration runner**

`internal/db/migrate.go`:

```go
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Migrate(d *sql.DB) error {
	if _, err := d.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			name       TEXT NOT NULL,
			applied_at TEXT NOT NULL
		)`); err != nil {
		return err
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}

		var exists int
		if err := d.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version,
		).Scan(&exists); err != nil {
			return err
		}
		if exists > 0 {
			continue
		}

		body, err := migrationsFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return err
		}

		tx, err := d.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, datetime('now'))`,
			version, name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func migrationVersion(filename string) (int, error) {
	prefix, _, ok := strings.Cut(filename, "_")
	if !ok {
		return 0, fmt.Errorf("migration %q: expected <version>_<name>.sql", filename)
	}
	return strconv.Atoi(prefix)
}
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/db/ -v`
Expected: PASS for `TestMigrateIsIdempotent` and `TestSchemaHasCoreTables`.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: add SQLite connection, migration runner, and core schema"
```

---

### Task 3: Seed content migration

**Files:**
- Create: `internal/db/migrations/0002_seed.sql`
- Create: `internal/db/seed_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate`.
- Produces: a database whose `profile`, `social_link`, `project`, `experience`, `skill`, `settings`, and `theme` tables hold the current site's content.

- [ ] **Step 1: Write the failing seed test**

`internal/db/seed_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"
)

func seeded(t *testing.T) *sql.DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return d
}

func count(t *testing.T, d *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestSeedCounts(t *testing.T) {
	d := seeded(t)

	cases := []struct {
		table string
		want  int
	}{
		{"profile", 1},
		{"social_link", 4},
		{"project", 9},
		{"experience", 8},
		{"skill", 30},
		{"theme", 1},
	}
	for _, c := range cases {
		if got := count(t, d, c.table); got != c.want {
			t.Errorf("%s count = %d, want %d", c.table, got, c.want)
		}
	}
}

func TestSeedProfileMatchesCurrentSite(t *testing.T) {
	d := seeded(t)

	var name, handle, location, dob string
	err := d.QueryRow(
		`SELECT name, handle, location, dob FROM profile WHERE id = 1`,
	).Scan(&name, &handle, &location, &dob)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}

	if name != "Morten Johansen" {
		t.Errorf("name = %q", name)
	}
	if handle != "mojoaar" {
		t.Errorf("handle = %q", handle)
	}
	if location != "Denmark" {
		t.Errorf("location = %q", location)
	}
	if dob != "1980-08-13" {
		t.Errorf("dob = %q", dob)
	}
}

func TestSeedProjectOrderAndHomelabIsNotALink(t *testing.T) {
	d := seeded(t)

	rows, err := d.Query(`
		SELECT name, is_link, url_label FROM project ORDER BY sort`)
	if err != nil {
		t.Fatalf("projects: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name, urlLabel string
		var isLink int
		if err := rows.Scan(&name, &isLink, &urlLabel); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
		if name == "homelab" {
			if isLink != 0 {
				t.Error("homelab is_link = 1, want 0")
			}
			if urlLabel != "self-hosted // private" {
				t.Errorf("homelab url_label = %q", urlLabel)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"atlascmdb", "clutch", "echo", "homelab", "icloud-mailflow",
		"ignite", "kanzo", "krypt", "mindmatrix",
	}
	if len(names) != len(want) {
		t.Fatalf("got %d projects, want %d", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("project[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestSeedSocialLinksAndSkills(t *testing.T) {
	d := seeded(t)

	rows, err := d.Query(`SELECT platform FROM social_link ORDER BY sort`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		platforms = append(platforms, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	want := []string{"bluesky", "linkedin", "mastodon", "github"}
	for i, w := range want {
		if i >= len(platforms) || platforms[i] != w {
			t.Fatalf("social platforms = %v, want %v", platforms, want)
		}
	}

	var firstName, lastName string
	if err := d.QueryRow(`SELECT name FROM skill ORDER BY sort LIMIT 1`).Scan(&firstName); err != nil {
		t.Fatal(err)
	}
	if firstName != "ITSM" {
		t.Errorf("first skill = %q, want ITSM", firstName)
	}
	if err := d.QueryRow(`SELECT name FROM skill ORDER BY sort DESC LIMIT 1`).Scan(&lastName); err != nil {
		t.Fatal(err)
	}
	if lastName != "Presenting" {
		t.Errorf("last skill = %q, want Presenting", lastName)
	}
}

func TestSeedSettingsDefaults(t *testing.T) {
	d := seeded(t)

	want := map[string]string{
		"posts_enabled":        "true",
		"active_theme":         "johansen",
		"stats_enabled":        "true",
		"stats_retention_days": "90",
		"timezone":             "Europe/Copenhagen",
		"site_title":           "Morten Johansen | johansen.foo",
		"seo_description":      "Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years. Global ops leader, open-source tinkerer, and automation enthusiast.",
	}
	for key, wantVal := range want {
		var got string
		if err := d.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&got); err != nil {
			t.Errorf("setting %q missing: %v", key, err)
			continue
		}
		if got != wantVal {
			t.Errorf("setting %q = %q, want %q", key, got, wantVal)
		}
	}
}
```

Add `"database/sql"` to the imports.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/db/ -run TestSeed -v`
Expected: FAIL — counts are `0, want 4` and similar.

- [ ] **Step 3: Write the seed migration**

`internal/db/migrations/0002_seed.sql`. Insert the profile with the hero bio and about paragraphs as **markdown**; the hero bio's JYSK link is markdown because the current page renders it as an anchor.

```sql
INSERT INTO profile (id, name, handle, location, dob, tagline, hero_bio, about_para_1, about_para_2, avatar, updated_at)
VALUES (
    1,
    'Morten Johansen',
    'mojoaar',
    'Denmark',
    '1980-08-13',
    '// enterprise it leader & open-source tinkerer',
    'Building and running complex infrastructure & cloud environments for 18+ years. Spent the last 5 leading global ops @ [JYSK](https://jysk.com) - 15 people, real budgets, and the quiet satisfaction of systems that heal themselves. Automation, architecture, and a few hard-earned lessons hold it all together. Father, husband, open-source tinkerer by night, with the occasional side project that scratches an itch.',
    'Two decades in IT — from Lance Corporal to global ops leader. I''ve sat on both sides of every ticket, outage, and migration. What I''ve learned: good infrastructure is boring by design, automation is cheaper than burnout, and the best IT leaders still remember what it felt like to be on call at 3 AM.',
    'Process that serves people, not the other way around. Clarity beats chaos every time.',
    'avatar.png',
    datetime('now')
);

INSERT INTO social_link (platform, url, label, sort) VALUES
    ('bluesky',  'https://bsky.app/profile/johansen.foo', 'Bluesky',  0),
    ('linkedin', 'https://linkedin.com/in/mojoaar',       'LinkedIn', 1),
    ('mastodon', 'https://floss.social/@mojoaar',         'Mastodon', 2),
    ('github',   'https://github.com/mojoaar',            'GitHub',   3);

INSERT INTO project (name, url, description, icon, is_link, url_label, sort) VALUES
    ('atlascmdb', 'https://github.com/mojoaar/atlascmdb',
     'Open-source CMDB built on Next.js - entity management, relationship graphing, TOTP MFA, SSO/SCIM, rack layouts, and bulk import.',
     'database', 1, 'github.com/mojoaar/atlascmdb', 0),
    ('clutch', 'https://github.com/mojoaar/clutch',
     'Cross-platform desktop AI chat - multi-provider streaming, markdown, file attachments, web fetching, 8 themes, and i18n. Tauri v2 + SvelteKit.',
     'message-square', 1, 'github.com/mojoaar/clutch', 1),
    ('echo', 'https://echo.johansen.foo',
     'What you look like from the internet''s perspective - WAN IP, ISP, location, timezone and more. No server. No tracking. Pure client-side.',
     'globe', 1, 'echo.johansen.foo', 2),
    ('homelab', NULL,
     'Four-node Proxmox VE 9 cluster (dagobah, geonosis, mendavi, serenno) running Ceph storage, SDN, HA, and a mix of LXC containers and QEMU VMs. Because every good automation starts at home.',
     'server', 0, 'self-hosted // private', 3),
    ('icloud-mailflow', 'https://github.com/mojoaar/icloud-mailflow',
     'IMAP rules engine for iCloud Mail - AND/OR logic, 13 operators, auto-reply, webhooks, dry-run, and MCP server. Go + HTMX.',
     'mail', 1, 'github.com/mojoaar/icloud-mailflow', 4),
    ('ignite', 'https://ignite.johansen.foo',
     'Provisioning with a heartbeat - AI-guided conversational interviews that generate project specs, agent guides, implementation plans, and READMEs. Wails desktop GUI.',
     'flame', 1, 'ignite.johansen.foo', 5),
    ('kanzo', 'https://github.com/mojoaar/kanzo',
     'Kanban board with configurable columns, drag & drop, GitHub sync, people & category systems, 13 themes, and full PWA support.',
     'kanban', 1, 'github.com/mojoaar/kanzo', 6),
    ('krypt', 'https://krypt.johansen.foo',
     'Terminal password manager with AES-256-GCM encryption, Argon2id key derivation, optional 2FA, and GitHub Gist sync. Built with Bubble Tea.',
     'shield-check', 1, 'krypt.johansen.foo', 7),
    ('mindmatrix', 'https://mindmatrix.johansen.foo',
     'Markdown-first, self-hosted, multi-user knowledge hub - realtime collaboration, backlinks, version history, plugins, and full REST API.',
     'brain', 1, 'mindmatrix.johansen.foo', 8);

INSERT INTO experience (years, role, company, icon, sort) VALUES
    ('2022 – present', 'Team Manager, IT Server Operations',              'JYSK',                  'briefcase',       0),
    ('2020 – 2022',    'Team Leader, IT Server Operations Nordic',        'JYSK',                  'briefcase',       1),
    ('2018 – 2020',    'Senior Consultant, ServiceNow',                   'Devoteam',              'square-terminal', 2),
    ('2016 – 2018',    'Systems Consultant, ServiceNow & Azure',          'Syspeople ApS',         'square-terminal', 3),
    ('2015 – 2016',    'Automation Engineer',                             'Wolseley',              'square-terminal', 4),
    ('2013 – 2015',    'Enterprise Systems Management Administrator',     'Wolseley',              'square-terminal', 5),
    ('2007 – 2013',    'Senior Technical Analyst',                        'Wolseley',              'square-terminal', 6),
    ('1999 – 2007',    'Lance Corporal, Tank Squadron',                   'Jydske Dragonregiment', 'shield',          7);

INSERT INTO skill (name, sort) VALUES
    ('ITSM', 0), ('ESM', 1), ('ServiceNow', 2), ('Jira', 3), ('ITIL', 4),
    ('Azure', 5), ('Google Cloud', 6), ('VMware', 7), ('KVM', 8), ('Proxmox', 9),
    ('Terraform', 10), ('OpenTofu', 11), ('IaC', 12), ('Automation', 13),
    ('Scripting', 14), ('Python', 15), ('Go', 16), ('PowerShell', 17),
    ('Active Directory', 18), ('Entra ID', 19), ('Linux', 20), ('Windows', 21),
    ('MacOS', 22), ('AI', 23), ('Leadership', 24), ('Management', 25),
    ('Budget', 26), ('Hosting', 27), ('Web Development', 28), ('Presenting', 29);

INSERT INTO settings (key, value) VALUES
    ('posts_enabled',        'true'),
    ('active_theme',         'johansen'),
    ('stats_enabled',        'true'),
    ('stats_retention_days', '90'),
    ('timezone',             'Europe/Copenhagen'),
    ('site_title',           'Morten Johansen | johansen.foo'),
    ('title_template',       '%s | johansen.foo'),
    ('seo_description',      'Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years. Global ops leader, open-source tinkerer, and automation enthusiast.'),
    ('og_image_url',         'https://johansen.foo/avatar.png'),
    ('og_type',              'website'),
    ('twitter_card',         'summary'),
    ('canonical_base_url',   'https://johansen.foo'),
    ('noindex',              'false'),
    ('sitemap_enabled',      'true'),
    ('robots_txt',           'User-agent: *\nAllow: /\n\nSitemap: https://johansen.foo/sitemap.xml');
```

The `theme` row is inserted in Task 5, once the token values are defined.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/db/ -v`
Expected: `TestSeed*` PASS. `TestSeedCounts` still expects `theme` count 1, so it will FAIL until Task 5. That is acceptable and expected — note it and move on.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: seed database with current site content"
```

---

### Task 4: Repositories

**Files:**
- Create: `internal/db/models.go`, `internal/db/repo_profile.go`, `internal/db/repo_content.go`, `internal/db/repo_settings.go`, `internal/db/repo_theme.go`
- Create: `internal/db/repo_test.go`

**Interfaces:**
- Consumes: `db.Open`, `db.Migrate`, the seeded schema.
- Produces:
  - `Models` — `Profile`, `SocialLink`, `Project`, `Experience`, `Skill`, `Theme`, `SiteContent`
  - `NewProfileRepo(*sql.DB) *ProfileRepo` with `Get() (*Profile, error)` and `SocialLinks() ([]SocialLink, error)`
  - `NewContentRepo(*sql.DB) *ContentRepo` with `Projects()`, `Experience()`, `Skills()` returning `([]T, error)`
  - `NewSettingsRepo(*sql.DB) *SettingsRepo` with `Get(key string) (string, error)` and `GetBool(key string) (bool, error)`
  - `NewThemeRepo(*sql.DB) *ThemeRepo` with `GetBySlug(slug string) (*Theme, error)`

- [ ] **Step 1: Define the models**

`internal/db/models.go`:

```go
package db

type Profile struct {
	Name       string
	Handle     string
	Location   string
	DOB        string
	Tagline    string
	HeroBio    string
	AboutPara1 string
	AboutPara2 string
	Avatar     string
}

type SocialLink struct {
	ID       int64
	Platform string
	URL      string
	Label    string
	Sort     int
}

type Project struct {
	ID          int64
	Name        string
	URL         string
	Description string
	Icon        string
	IsLink      bool
	URLLabel    string
	Sort        int
}

type Experience struct {
	ID      int64
	Years   string
	Role    string
	Company string
	Icon    string
	Sort    int
}

type Skill struct {
	ID   int64
	Name string
	Sort int
}

type Theme struct {
	ID          int64
	Slug        string
	Name        string
	Description string
	TokensBase  map[string]string
	TokensLight map[string]string
	TokensDark  map[string]string
}

type SiteContent struct {
	Profile    Profile
	Social     []SocialLink
	Projects   []Project
	Experience []Experience
	Skills     []Skill
	Theme      Theme
	Settings   map[string]string
}
```

- [ ] **Step 2: Write the failing repository test**

`internal/db/repo_test.go`:

```go
package db

import "testing"

func TestProfileRepoGet(t *testing.T) {
	d := seeded(t)

	p, err := NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "Morten Johansen" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.HeroBio == "" || p.AboutPara1 == "" {
		t.Error("bio fields are empty")
	}
}

func TestContentRepoOrdering(t *testing.T) {
	d := seeded(t)
	repo := NewContentRepo(d)

	projects, err := repo.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 9 {
		t.Fatalf("got %d projects, want 9", len(projects))
	}
	if projects[3].Name != "homelab" || projects[3].IsLink {
		t.Errorf("projects[3] = %+v, want non-link homelab", projects[3])
	}

	exp, err := repo.Experience()
	if err != nil {
		t.Fatalf("Experience: %v", err)
	}
	if len(exp) != 8 {
		t.Fatalf("got %d experience rows, want 8", len(exp))
	}
	if exp[0].Company != "JYSK" || exp[7].Company != "Jydske Dragonregiment" {
		t.Errorf("experience order wrong: first=%q last=%q", exp[0].Company, exp[7].Company)
	}

	skills, err := repo.Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	if len(skills) != 30 {
		t.Fatalf("got %d skills, want 30", len(skills))
	}
	if skills[0].Name != "ITSM" || skills[29].Name != "Presenting" {
		t.Errorf("skill order wrong: first=%q last=%q", skills[0].Name, skills[29].Name)
	}
}

func TestSettingsRepo(t *testing.T) {
	d := seeded(t)
	repo := NewSettingsRepo(d)

	got, err := repo.Get("active_theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "johansen" {
		t.Errorf("active_theme = %q, want johansen", got)
	}

	on, err := repo.GetBool("posts_enabled")
	if err != nil {
		t.Fatalf("GetBool: %v", err)
	}
	if !on {
		t.Error("posts_enabled = false, want true")
	}
}

func TestSettingsRepoMissingKeyErrors(t *testing.T) {
	d := seeded(t)
	if _, err := NewSettingsRepo(d).Get("does_not_exist"); err == nil {
		t.Fatal("Get on missing key succeeded, want error")
	}
}
```

Note: `TestSettingsRepo` and `TestContentRepoOrdering` will not compile until the repos exist.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/db/ -run 'TestProfileRepo|TestContentRepo|TestSettingsRepo' -v`
Expected: FAIL — `undefined: NewProfileRepo`.

- [ ] **Step 4: Implement the repositories**

`internal/db/repo_profile.go`:

```go
package db

import "database/sql"

type ProfileRepo struct{ db *sql.DB }

func NewProfileRepo(d *sql.DB) *ProfileRepo { return &ProfileRepo{db: d} }

func (r *ProfileRepo) Get() (*Profile, error) {
	var p Profile
	err := r.db.QueryRow(`
		SELECT name, handle, location, dob, tagline, hero_bio,
		       about_para_1, about_para_2, avatar
		FROM profile WHERE id = 1`).
		Scan(&p.Name, &p.Handle, &p.Location, &p.DOB, &p.Tagline,
			&p.HeroBio, &p.AboutPara1, &p.AboutPara2, &p.Avatar)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepo) SocialLinks() ([]SocialLink, error) {
	rows, err := r.db.Query(`
		SELECT id, platform, url, label, sort
		FROM social_link WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SocialLink
	for rows.Next() {
		var s SocialLink
		if err := rows.Scan(&s.ID, &s.Platform, &s.URL, &s.Label, &s.Sort); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
```

`internal/db/repo_content.go`:

```go
package db

import "database/sql"

type ContentRepo struct{ db *sql.DB }

func NewContentRepo(d *sql.DB) *ContentRepo { return &ContentRepo{db: d} }

func (r *ContentRepo) Projects() ([]Project, error) {
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(url, ''), description, icon, is_link, url_label, sort
		FROM project WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Description,
			&p.Icon, &p.IsLink, &p.URLLabel, &p.Sort); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Experience() ([]Experience, error) {
	rows, err := r.db.Query(`
		SELECT id, years, role, company, icon, sort
		FROM experience WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Experience
	for rows.Next() {
		var e Experience
		if err := rows.Scan(&e.ID, &e.Years, &e.Role, &e.Company, &e.Icon, &e.Sort); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Skills() ([]Skill, error) {
	rows, err := r.db.Query(`
		SELECT id, name, sort FROM skill WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.Sort); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
```

`internal/db/repo_settings.go`:

```go
package db

import (
	"database/sql"
	"errors"
	"strconv"
)

var ErrSettingNotFound = errors.New("setting not found")

type SettingsRepo struct{ db *sql.DB }

func NewSettingsRepo(d *sql.DB) *SettingsRepo { return &SettingsRepo{db: d} }

func (r *SettingsRepo) Get(key string) (string, error) {
	var v string
	err := r.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrSettingNotFound
	}
	return v, err
}

func (r *SettingsRepo) GetBool(key string) (bool, error) {
	v, err := r.Get(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(v)
}
```

`internal/db/repo_theme.go`:

```go
package db

import (
	"database/sql"
	"encoding/json"
)

type ThemeRepo struct{ db *sql.DB }

func NewThemeRepo(d *sql.DB) *ThemeRepo { return &ThemeRepo{db: d} }

func (r *ThemeRepo) GetBySlug(slug string) (*Theme, error) {
	var t Theme
	var baseRaw, lightRaw, darkRaw string

	err := r.db.QueryRow(`
		SELECT id, slug, name, description, tokens_base, tokens_light, tokens_dark
		FROM theme WHERE slug = ?`, slug).
		Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &baseRaw, &lightRaw, &darkRaw)
	if err != nil {
		return nil, err
	}

	for _, f := range []struct {
		raw  string
		into *map[string]string
	}{
		{baseRaw, &t.TokensBase},
		{lightRaw, &t.TokensLight},
		{darkRaw, &t.TokensDark},
	} {
		if err := json.Unmarshal([]byte(f.raw), f.into); err != nil {
			return nil, err
		}
	}
	return &t, nil
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/db/ -v`
Expected: the repo tests PASS. `TestSeedCounts` still fails on `theme`.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: add repositories for profile, content, settings, and themes"
```

---

### Task 5: Theme package and the `johansen` base theme

**Files:**
- Create: `internal/theme/theme.go`, `internal/theme/css.go`, `internal/theme/johansen.go`
- Create: `internal/theme/theme_test.go`
- Create: `internal/db/migrations/0003_johansen_theme.sql`

**Interfaces:**
- Consumes: `db.Theme`.
- Produces:
  - `theme.Theme{Slug string, Base, Light, Dark map[string]string}`
  - `theme.Johansen() Theme`
  - `theme.Validate(t Theme) error`
  - `theme.CSS(t Theme) string`
  - `theme.Vars(theme map[string]string) string` — for tests and `/api/v1/theme`

The token set Phase 1 supports is exactly the set the ported `style.css` consumes: `--bg`, `--bg2`, `--bg3`, `--border`, `--text`, `--text-muted`, `--accent`, `--accent2`, `--accent-glow`, `--green`, `--shadow`, `--radius`, `--font-mono`, `--font-sans`.

- [ ] **Step 1: Write the failing theme test**

`internal/theme/theme_test.go`:

```go
package theme

import (
	"strings"
	"testing"
)

func TestJohansenHasEveryToken(t *testing.T) {
	th := Johansen()

	if err := Validate(th); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, mode := range []map[string]string{th.Base, th.Light, th.Dark} {
		for name := range mode {
			if !strings.HasPrefix(name, "--") {
				t.Errorf("token %q is not a CSS custom property", name)
			}
		}
	}
}

func TestJohansenDarkValuesMatchCurrentSite(t *testing.T) {
	th := Johansen()

	want := map[string]string{
		"--bg":         "#0f1117",
		"--bg2":        "#161b27",
		"--bg3":        "#1e2433",
		"--border":     "#2a3045",
		"--text":       "#e2e8f0",
		"--text-muted": "#8892a4",
		"--accent":     "#7c6af7",
		"--accent2":    "#a78bfa",
		"--green":      "#34d399",
	}
	for name, wantVal := range want {
		if got := th.Dark[name]; got != wantVal {
			t.Errorf("dark %s = %q, want %q", name, got, wantVal)
		}
	}
}

func TestJohansenLightValuesMatchCurrentSite(t *testing.T) {
	th := Johansen()

	want := map[string]string{
		"--bg":         "#f4f6fb",
		"--bg2":        "#ffffff",
		"--bg3":        "#eef1f8",
		"--border":     "#d0d7e3",
		"--text":       "#1a1f2e",
		"--text-muted": "#5a6278",
		"--accent":     "#5b4edc",
		"--accent2":    "#7c6af7",
		"--green":      "#059669",
	}
	for name, wantVal := range want {
		if got := th.Light[name]; got != wantVal {
			t.Errorf("light %s = %q, want %q", name, got, wantVal)
		}
	}
}

func TestValidateRejectsUnknownToken(t *testing.T) {
	th := Johansen()
	th.Dark["--not-a-real-token"] = "#fff"

	if err := Validate(th); err == nil {
		t.Fatal("Validate accepted an unknown token, want error")
	}
}

func TestValidateRejectsMissingToken(t *testing.T) {
	th := Johansen()
	delete(th.Dark, "--bg")

	if err := Validate(th); err == nil {
		t.Fatal("Validate accepted a theme missing --bg in dark, want error")
	}
}

func TestCSSScopesByThemeAndMode(t *testing.T) {
	css := CSS(Johansen())

	if !strings.Contains(css, `[data-theme="johansen"]`) {
		t.Error("CSS does not scope to the theme slug")
	}
	if !strings.Contains(css, `[data-theme="johansen"][data-mode="dark"]`) {
		t.Error("CSS does not scope dark mode")
	}
	if !strings.Contains(css, `[data-theme="johansen"][data-mode="light"]`) {
		t.Error("CSS does not scope light mode")
	}
	if !strings.Contains(css, "--bg:#0f1117") {
		t.Error("CSS does not contain the dark --bg value")
	}
	if !strings.Contains(css, "--bg:#f4f6fb") {
		t.Error("CSS does not contain the light --bg value")
	}
	if strings.Contains(css, "--radius") && strings.Count(css, "--radius:") != 1 {
		t.Error("--radius should be emitted once, in the base block")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/theme/ -v`
Expected: FAIL — `undefined: Johansen`.

- [ ] **Step 3: Implement the vocabulary and validation**

`internal/theme/theme.go`:

```go
package theme

import (
	"fmt"
	"sort"
)

type Theme struct {
	Slug        string
	Name        string
	Description string
	Base        map[string]string
	Light       map[string]string
	Dark        map[string]string
}

var requiredTokens = []string{
	"--bg", "--bg2", "--bg3", "--border",
	"--text", "--text-muted",
	"--accent", "--accent2", "--accent-glow", "--green",
	"--shadow",
}

var optionalTokens = []string{
	"--radius", "--font-mono", "--font-sans",
}

func known() map[string]bool {
	m := make(map[string]bool, len(requiredTokens)+len(optionalTokens))
	for _, t := range append(append([]string{}, requiredTokens...), optionalTokens...) {
		m[t] = true
	}
	return m
}

func Validate(t Theme) error {
	knownTokens := known()

	for _, section := range []struct {
		label  string
		tokens map[string]string
	}{
		{"base", t.Base},
		{"light", t.Light},
		{"dark", t.Dark},
	} {
		for name := range section.tokens {
			if !knownTokens[name] {
				return fmt.Errorf("theme %q: unknown token %q in %s", t.Slug, name, section.label)
			}
		}
	}

	for _, name := range requiredTokens {
		if t.Light[name] == "" {
			return fmt.Errorf("theme %q: missing %s in light", t.Slug, name)
		}
		if t.Dark[name] == "" {
			return fmt.Errorf("theme %q: missing %s in dark", t.Slug, name)
		}
	}
	return nil
}

func Resolve(base, mode map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(mode))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range mode {
		out[k] = v
	}
	return out
}

func Vars(tokens map[string]string) string {
	names := make([]string, 0, len(tokens))
	for name := range tokens {
		names = append(names, name)
	}
	sort.Strings(names)

	out := ""
	for i, name := range names {
		if i > 0 {
			out += ";"
		}
		out += name + ":" + tokens[name]
	}
	return out
}
```

- [ ] **Step 4: Implement CSS emission**

`internal/theme/css.go`:

```go
package theme

import "strings"

func CSS(t Theme) string {
	var b strings.Builder

	if len(t.Base) > 0 {
		b.WriteString(`[data-theme="` + t.Slug + `"]{`)
		b.WriteString(Vars(t.Base))
		b.WriteString("}\n")
	}

	for _, mode := range []struct {
		name   string
		tokens map[string]string
	}{
		{"light", t.Light},
		{"dark", t.Dark},
	} {
		if len(mode.tokens) == 0 {
			continue
		}
		b.WriteString(`[data-theme="` + t.Slug + `"][data-mode="` + mode.name + `"]{`)
		b.WriteString(Vars(mode.tokens))
		b.WriteString("}\n")
	}

	return b.String()
}
```

- [ ] **Step 5: Define the `johansen` theme**

`internal/theme/johansen.go`:

```go
package theme

func Johansen() Theme {
	return Theme{
		Slug:        "johansen",
		Name:        "Johansen",
		Description: "The original johansen.foo theme.",
		Base: map[string]string{
			"--radius": "10px",
			"--font-mono": `"JetBrains Mono", "Fira Code", "Cascadia Code", ui-monospace, monospace`,
			"--font-sans": `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif`,
		},
		Dark: map[string]string{
			"--bg":         "#0f1117",
			"--bg2":        "#161b27",
			"--bg3":        "#1e2433",
			"--border":     "#2a3045",
			"--text":       "#e2e8f0",
			"--text-muted": "#8892a4",
			"--accent":     "#7c6af7",
			"--accent2":    "#a78bfa",
			"--accent-glow": "rgba(124, 106, 247, 0.25)",
			"--green":      "#34d399",
			"--shadow":     "0 4px 32px rgba(0, 0, 0, 0.5)",
		},
		Light: map[string]string{
			"--bg":         "#f4f6fb",
			"--bg2":        "#ffffff",
			"--bg3":        "#eef1f8",
			"--border":     "#d0d7e3",
			"--text":       "#1a1f2e",
			"--text-muted": "#5a6278",
			"--accent":     "#5b4edc",
			"--accent2":    "#7c6af7",
			"--accent-glow": "rgba(91, 78, 220, 0.15)",
			"--green":      "#059669",
			"--shadow":     "0 4px 24px rgba(0, 0, 0, 0.08)",
		},
	}
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/theme/ -v`
Expected: PASS.

- [ ] **Step 7: Write the theme seed migration**

`internal/db/migrations/0003_johansen_theme.sql`. The JSON must match `theme.Johansen()` exactly — Step 8 adds `TestSeededThemeMatchesBuiltin` to the `db` package to assert this.

```sql
INSERT INTO theme (slug, name, description, tokens_base, tokens_light, tokens_dark, sort, created_at, updated_at)
VALUES (
    'johansen',
    'Johansen',
    'The original johansen.foo theme.',
    '{"--font-mono":"\"JetBrains Mono\", \"Fira Code\", \"Cascadia Code\", ui-monospace, monospace","--font-sans":"-apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, sans-serif","--radius":"10px"}',
    '{"--accent":"#5b4edc","--accent-glow":"rgba(91, 78, 220, 0.15)","--accent2":"#7c6af7","--bg":"#f4f6fb","--bg2":"#ffffff","--bg3":"#eef1f8","--border":"#d0d7e3","--green":"#059669","--shadow":"0 4px 24px rgba(0, 0, 0, 0.08)","--text":"#1a1f2e","--text-muted":"#5a6278"}',
    '{"--accent":"#7c6af7","--accent-glow":"rgba(124, 106, 247, 0.25)","--accent2":"#a78bfa","--bg":"#0f1117","--bg2":"#161b27","--bg3":"#1e2433","--border":"#2a3045","--green":"#34d399","--shadow":"0 4px 32px rgba(0, 0, 0, 0.5)","--text":"#e2e8f0","--text-muted":"#8892a4"}',
    0,
    datetime('now'),
    datetime('now')
);
```

- [ ] **Step 8: Add the cross-check test**

Append to `internal/db/repo_test.go`:

```go
func TestSeededThemeMatchesBuiltin(t *testing.T) {
	d := seeded(t)

	got, err := NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}

	want := theme.Johansen()

	for _, c := range []struct {
		label string
		got   map[string]string
		want  map[string]string
	}{
		{"base", got.TokensBase, want.Base},
		{"light", got.TokensLight, want.Light},
		{"dark", got.TokensDark, want.Dark},
	} {
		if len(c.got) != len(c.want) {
			t.Errorf("%s: got %d tokens, want %d", c.label, len(c.got), len(c.want))
		}
		for k, v := range c.want {
			if c.got[k] != v {
				t.Errorf("%s %s = %q, want %q", c.label, k, c.got[k], v)
			}
		}
	}
}
```

Add `"github.com/mojoaar/johansenfoo/internal/theme"` to that file's imports.

- [ ] **Step 9: Run the full suite**

Run: `go test ./... -v`
Expected: all PASS, including `TestSeedCounts` (theme count is now 1).

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: add theme package with token vocabulary and the johansen base theme"
```

---

### Task 6: Icons package

**Files:**
- Create: `internal/icons/svg/*.svg` (vendored)
- Create: `internal/icons/icons.go`, `internal/icons/icons_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `icons.Inline(name, class string) template.HTML`, `icons.Exists(name string) bool`.

- [ ] **Step 1: Vendor the icon sources**

Lucide provides the outline icons the site already uses (it is the same icon set the old page loaded from the CDN), and Simple Icons covers the social brands that Font Awesome was providing.

```bash
cd /Users/mojoaar/Development/johansenfoo
mkdir -p internal/icons/svg
LUCIDE=https://unpkg.com/lucide-static@0.503.0/icons
for n in database message-square globe server mail flame kanban shield-check brain \
         briefcase square-terminal shield arrow-up-right-from-square; do
  curl -sSfL "$LUCIDE/$n.svg" -o "internal/icons/svg/$n.svg"
done

SIMPLE=https://unpkg.com/simple-icons@15.0.0/icons
for n in bluesky linkedin mastodon github; do
  curl -sSfL "$SIMPLE/$n.svg" -o "internal/icons/svg/$n.svg"
done
ls internal/icons/svg
```

Expected: 17 `.svg` files. Verify each is non-empty and starts with `<svg`:

```bash
head -c 40 internal/icons/svg/globe.svg; echo
```

Record the provenance and licences in `internal/icons/README.md`: Lucide is ISC, Simple Icons is CC0-1.0. Note the pinned versions.

- [ ] **Step 2: Write the failing icons test**

`internal/icons/icons_test.go`:

```go
package icons

import (
	"strings"
	"testing"
)

func TestEveryIconNameResolves(t *testing.T) {
	names := []string{
		"database", "message-square", "globe", "server", "mail", "flame",
		"kanban", "shield-check", "brain", "briefcase", "square-terminal",
		"shield", "arrow-up-right-from-square",
		"bluesky", "linkedin", "mastodon", "github",
	}
	for _, name := range names {
		if !Exists(name) {
			t.Errorf("icon %q is not vendored", name)
			continue
		}
		svg := string(Inline(name, "project-icon"))
		if !strings.HasPrefix(svg, "<svg") {
			t.Errorf("icon %q does not start with <svg: %q", name, svg)
		}
		if !strings.Contains(svg, `class="project-icon"`) {
			t.Errorf("icon %q is missing the supplied class", name)
		}
		if strings.Contains(svg, "width=") || strings.Contains(svg, "height=") {
			t.Errorf("icon %q still carries fixed dimensions", name)
		}
	}
}

func TestUnknownIconReturnsEmpty(t *testing.T) {
	if got := Inline("no-such-icon", "x"); got != "" {
		t.Errorf("Inline(unknown) = %q, want empty", got)
	}
}

func TestInlineSetsAccessibilityAttributes(t *testing.T) {
	svg := string(Inline("globe", "project-icon"))
	if !strings.Contains(svg, `aria-hidden="true"`) {
		t.Error("icon is missing aria-hidden")
	}
	if !strings.Contains(svg, `focusable="false"`) {
		t.Error("icon is missing focusable=false")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/icons/ -v`
Expected: FAIL — `undefined: Exists`.

- [ ] **Step 4: Implement the package**

`internal/icons/icons.go`:

```go
package icons

import (
	"embed"
	"html/template"
	"path"
	"regexp"
	"strings"
	"sync"
)

//go:embed svg/*.svg
var svgFS embed.FS

var (
	dimensionRE = regexp.MustCompile(`\s(width|height)="[^"]*"`)
	classRE     = regexp.MustCompile(`\sclass="[^"]*"`)
	fillRE      = regexp.MustCompile(`\sfill="[^"]*"`)
	cache       sync.Map
)

func Exists(name string) bool {
	_, err := svgFS.ReadFile(path.Join("svg", name+".svg"))
	return err == nil
}

func Inline(name, class string) template.HTML {
	if v, ok := cache.Load(name); ok {
		return render(v.(string), class)
	}
	raw, err := svgFS.ReadFile(path.Join("svg", name+".svg"))
	if err != nil {
		return ""
	}
	body := string(raw)
	cache.Store(name, body)
	return render(body, class)
}

func render(body, class string) template.HTML {
	body = dimensionRE.ReplaceAllString(body, "")
	body = classRE.ReplaceAllString(body, "")
	body = fillRE.ReplaceAllStringFunc(body, func(match string) string {
		if strings.Contains(match, `"none"`) {
			return match
		}
		return ` fill="currentColor"`
	})

	attrs := ` class="` + class + `" aria-hidden="true" focusable="false"`
	if strings.HasPrefix(body, "<svg") {
		if i := strings.IndexByte(body, '>'); i > 0 {
			body = body[:i] + attrs + body[i:]
		}
	}
	return template.HTML(body)
}
```

The `fill` transform must preserve Lucide's `fill="none"` on the root element, or the outline icons would render as solid shapes. Go's regexp engine has no negative lookahead, hence the callback form.

Notes on the transforms: dropping `width`/`height` lets the stylesheet size the icon, which is how the current `.project-icon`/`.tl-icon` rules already work. Forcing `fill="currentColor"` on the Simple Icons brand glyphs makes them inherit the social link's colour, matching how the Font Awesome icons behaved. `template.HTML` is safe here because the content comes from our own embedded files, never from user input.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/icons/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: vendor icons and render them as inline SVG"
```

---

### Task 7: Static assets and the stylesheet port

**Files:**
- Create: `internal/web/static/style.css` (ported)
- Create: `internal/web/static/fonts/*.woff2`, `internal/web/static/fonts/OFL.txt`
- Copy: `favicon.svg`, `favicon-16.png`, `favicon-32.png`, `favicon.ico`, `apple-touch-icon.png`, `avatar.png`
- Create: `internal/web/static.go`, update `internal/web/web.go`, `internal/web/web_test.go`

**Interfaces:**
- Consumes: `web.New`.
- Produces: a filesystem served at `/static/*` and a `web.staticFS` handle the renderer uses for asset URLs.

- [ ] **Step 1: Copy the binary assets**

```bash
cd /Users/mojoaar/Development/johansenfoo
mkdir -p internal/web/static/fonts
SRC=/Users/mojoaar/Development/johansen_landing
cp "$SRC/avatar.png" "$SRC/favicon.svg" "$SRC/favicon-16.png" \
   "$SRC/favicon-32.png" "$SRC/favicon.ico" "$SRC/apple-touch-icon.png" \
   internal/web/static/
ls internal/web/static
```

Expected: `apple-touch-icon.png  avatar.png  favicon-16.png  favicon-32.png  favicon.ico  favicon.svg  fonts`.

- [ ] **Step 2: Self-host JetBrains Mono**

Fetch the Google Fonts CSS with a browser user agent so it returns woff2 URLs, then extract the weights with Python rather than reading them by eye:

```bash
cd /Users/mojoaar/Development/johansenfoo/internal/web/static/fonts
curl -sS -H 'User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36' \
  'https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;700&display=swap' \
  -o google.css

python3 - <<'PY'
import re
css = open('google.css').read()

blocks = re.findall(r'@font-face\s*\{[^}]*\}', css)
assert blocks, 'no @font-face blocks in google.css'

found = {}
for block in blocks:
    if 'U+0000-00FF' not in block:
        continue
    weight = re.search(r'font-weight:\s*(\d+)', block).group(1)
    url = re.search(r'url\((https://[^)]+\.woff2)\)', block)
    if url:
        found[weight] = url.group(1)

for weight in ('400', '700'):
    if weight not in found:
        raise SystemExit(f'missing latin woff2 for weight {weight}')

for weight, url in sorted(found.items()):
    open(f'url-{weight}.txt', 'w').write(url)
    print(f'{weight}\t{url}')
PY
```

Expected: two lines, `400` and `700`, each with a `fonts.gstatic.com` woff2 URL. Now download them and clean up:

```bash
cd /Users/mojoaar/Development/johansenfoo/internal/web/static/fonts
curl -sSfL "$(cat url-400.txt)" -o jetbrains-mono-latin-400.woff2
curl -sSfL "$(cat url-700.txt)" -o jetbrains-mono-latin-700.woff2
rm -f url-400.txt url-700.txt google.css

curl -sSfL https://raw.githubusercontent.com/JetBrains/JetBrainsMono/master/OFL.txt -o OFL.txt
ls -la
```

Expected: `jetbrains-mono-latin-400.woff2`, `jetbrains-mono-latin-700.woff2` (roughly 30–40 KB each), and `OFL.txt`. Verify the fonts are real woff2 files, not error pages:

```bash
file jetbrains-mono-latin-400.woff2 jetbrains-mono-latin-700.woff2
```

Expected: `Web Open Font Format (Version 2)`. A `text/html` result means the download failed and must be retried.

- [ ] **Step 3: Port the stylesheet**

Copy it, then apply exactly two edits.

```bash
cp /Users/mojoaar/Development/johansen_landing/style.css \
   /Users/mojoaar/Development/johansenfoo/internal/web/static/style.css
```

**Edit 1 — delete the token blocks, lines 1–35.** Everything from the opening `/* ─── Design tokens ─── */` comment through the closing brace of the `[data-theme="light"]` block is now emitted by the theme package. Wrap the `@font-face` declarations that Step 2 requires in their place:

```css
/* ─── Self-hosted fonts ─────────────────────────────────────────── */
@font-face {
    font-family: "JetBrains Mono";
    font-style: normal;
    font-weight: 400;
    font-display: swap;
    src: url("/static/fonts/jetbrains-mono-latin-400.woff2") format("woff2");
}

@font-face {
    font-family: "JetBrains Mono";
    font-style: normal;
    font-weight: 700;
    font-display: swap;
    src: url("/static/fonts/jetbrains-mono-latin-700.woff2") format("woff2");
}
```

**Edit 2 — the mode selector.** The light/dark axis moved from `data-theme` to `data-mode`, because `data-theme` now carries the theme slug. There are exactly two occurrences, currently at lines 164 and 167:

```css
/* before */
[data-theme="light"] .toggle-icon.moon {
[data-theme="light"] .toggle-icon.sun {

/* after */
[data-mode="light"] .toggle-icon.moon {
[data-mode="light"] .toggle-icon.sun {
```

Everything else in the file stays byte-for-byte identical. Verify the port is clean:

```bash
cd /Users/mojoaar/Development/johansenfoo
echo "--- must be empty: no leftover token blocks ---"
grep -nE '^\s*--(bg|text|accent|border|green|shadow|radius|font-)' internal/web/static/style.css
echo "--- must be empty: no data-theme light/dark selectors ---"
grep -n 'data-theme="light"\|data-theme="dark"' internal/web/static/style.css
echo "--- must be exactly 2 ---"
grep -c 'data-mode="light"' internal/web/static/style.css
echo "--- diff excluding the removed header and the two edits ---"
diff <(sed -n '36,581p' /Users/mojoaar/Development/johansen_landing/style.css) \
     <(sed -n '/Reset & base/,$p' internal/web/static/style.css) \
  && echo "BODY IDENTICAL (modulo the two intended edits)"
```

The final `diff` will report the two `data-mode` lines if the port is correct and nothing else. Confirm that is the only difference.

- [ ] **Step 4: Write the failing static-asset test**

Append to `internal/web/web_test.go`:

```go
func TestStaticStylesheetServed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
	body := rec.Body.String()
	if strings.Contains(body, ":root {") {
		t.Error("stylesheet still declares its own token block")
	}
	if !strings.Contains(body, `[data-mode="light"]`) {
		t.Error("stylesheet does not use the data-mode axis")
	}
}

func TestStaticAvatarServed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/avatar.png", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("avatar.png is empty")
	}
}
```

Add `"strings"` to the imports.

- [ ] **Step 5: Run the test to verify it fails**

Run: `go test ./internal/web/ -run TestStatic -v`
Expected: FAIL — 404.

- [ ] **Step 6: Implement the embedded static server**

`internal/web/static.go`:

```go
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:static
var staticFS embed.FS

func staticHandler() (http.Handler, error) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(sub))), nil
}
```

Register it in `internal/web/web.go` inside `New`, replacing the `r.Get("/health", ...)` block with:

```go
	static, err := staticHandler()
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", static)

	r.Get("/health", healthHandler(d))
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/web/ -v`
Expected: PASS.

- [ ] **Step 8: Confirm no third-party origins remain**

```bash
cd /Users/mojoaar/Development/johansenfoo
grep -rnE "fonts\.googleapis|fonts\.gstatic|cdnjs\.cloudflare|cdn\.jsdelivr" internal/web/static/ && echo "FOUND THIRD PARTY" || echo "CLEAN"
```

Expected: `CLEAN`.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: serve ported stylesheet and self-hosted assets from /static"
```

---

### Task 8: Landing page templates rendered from the database

**Files:**
- Create: `internal/markdown/markdown.go`
- Create: `internal/web/templates/base.html`, `internal/web/templates/landing.html`
- Create: `internal/web/render.go`, `internal/web/landing.go`
- Create: `internal/web/landing_test.go`

**Interfaces:**
- Consumes: `db.SiteContent`, `theme.CSS`, `icons.Inline`, `markdown.Render`.
- Produces: `markdown.Render(src string) template.HTML`; `web.New` gaining a `Content *db.SiteContent` field on `Deps` and serving `GET /`.

- [ ] **Step 1: Add goldmark and implement the markdown package**

```bash
go get github.com/yuin/goldmark@v1.7.13
go get github.com/yuin/goldmark/extension@v1.7.13
```

`internal/markdown/markdown.go`:

```go
package markdown

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
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
```

Phase 4 adds Chroma highlighting and a sanitiser to this package. It is deliberately minimal here — the only markdown in Phase 1 is seeded site copy.

- [ ] **Step 2: Write the base template**

`internal/web/templates/base.html`:

```html
{{define "base"}}<!doctype html>
<html lang="en" data-theme="{{.ThemeSlug}}" data-mode="dark">
    <head>
        <meta charset="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <title>{{.Meta.Title}}</title>
        <meta name="description" content="{{.Meta.Description}}" />
        <meta name="author" content="{{.Profile.Name}}" />
        <meta name="robots" content="{{.Meta.Robots}}" />
        <link rel="canonical" href="{{.Meta.Canonical}}" />

        <link rel="icon" type="image/svg+xml" href="/static/favicon.svg" />
        <link rel="icon" type="image/png" sizes="32x32" href="/static/favicon-32.png" />
        <link rel="icon" type="image/png" sizes="16x16" href="/static/favicon-16.png" />
        <link rel="apple-touch-icon" sizes="180x180" href="/static/apple-touch-icon.png" />
        <meta name="theme-color" content="{{.Meta.ThemeColor}}" />

        <meta property="og:type" content="{{.Meta.OGType}}" />
        <meta property="og:url" content="{{.Meta.Canonical}}" />
        <meta property="og:title" content="{{.Meta.Title}}" />
        <meta property="og:description" content="{{.Meta.Description}}" />
        <meta property="og:image" content="{{.Meta.OGImage}}" />
        <meta property="og:image:width" content="600" />
        <meta property="og:image:height" content="600" />
        <meta property="og:image:alt" content="{{.Profile.Name}}" />
        <meta property="og:locale" content="en_US" />
        <meta property="og:site_name" content="johansen.foo" />

        <meta name="twitter:card" content="{{.Meta.TwitterCard}}" />
        <meta name="twitter:title" content="{{.Meta.Title}}" />
        <meta name="twitter:description" content="{{.Meta.Description}}" />
        <meta name="twitter:image" content="{{.Meta.OGImage}}" />
        <meta name="twitter:image:alt" content="{{.Profile.Name}}" />

        {{.StructuredData}}

        <style>{{.ThemeCSS}}</style>
        <link rel="stylesheet" href="/static/style.css" />
        <script>
            (function () {
                var html = document.documentElement;
                var saved =
                    localStorage.getItem("mode") || localStorage.getItem("theme");
                if (saved === "light" || saved === "dark") {
                    html.setAttribute("data-mode", saved);
                } else if (
                    window.matchMedia("(prefers-color-scheme: light)").matches
                ) {
                    html.setAttribute("data-mode", "light");
                } else {
                    html.setAttribute("data-mode", "dark");
                }
            })();
        </script>
    </head>
    <body>
        {{template "content" .}}
    </body>
</html>
{{end}}
```

Notes: the IIFE still reads the old `theme` localStorage key as a fallback so returning visitors keep their choice across the migration, but writes only `mode` from here on. `{{.StructuredData}}` is injected as raw JSON-LD (typed `template.HTML` by the handler, since it is generated, not user data).

- [ ] **Step 3: Write the landing template**

`internal/web/templates/landing.html`, marked up to match the current page exactly:

```html
{{define "content"}}
        <nav aria-label="Main navigation">
            <a href="#home" class="nav-logo">johansen.foo</a>
            <button class="hamburger" id="hamburger" aria-label="Toggle navigation" aria-expanded="false" aria-controls="navLinks">&#9776;</button>
            <ul class="nav-links" id="navLinks" role="list">
                <li><a href="#home">Home</a></li>
                <li><a href="#projects">Projects</a></li>
                <li><a href="#about">About</a></li>
                <li>
                    <button class="theme-toggle" id="themeToggle" aria-label="Toggle colour scheme" title="Toggle light/dark mode">
                        <span class="toggle-icon moon" aria-hidden="true">🌙</span>
                        <span class="toggle-icon sun" aria-hidden="true">☀️</span>
                    </button>
                </li>
            </ul>
        </nav>

        <section class="hero" id="home">
            <div class="container">
                <img src="/static/avatar.png" alt="{{.Profile.Name}}" class="hero-avatar" />
                <h1 class="hero-name">{{.Profile.Name}}</h1>
                <p class="hero-title">{{.Profile.Tagline}}</p>
                <p class="hero-bio">{{.HeroBio}}</p>
                <div class="hero-social">
                    {{range .Social}}
                    <a href="{{.URL}}" target="_blank" rel="noopener noreferrer" aria-label="{{.Label}}" class="social-{{.Platform}}">{{icon .Platform ""}}</a>
                    {{end}}
                </div>
            </div>
        </section>

        <section id="projects">
            <div class="container">
                <p class="section-label">// projects</p>
                <h2 class="section-title">Things I've built</h2>
                <p class="section-desc">
                    A couple of side projects that live under the
                    <code class="inline-code">johansen.foo</code> umbrella.
                </p>

                <div class="projects-grid">
                    {{range .Projects}}
                    {{if .IsLink}}
                    <a href="{{.URL}}" target="_blank" rel="noopener noreferrer" class="project-card is-link">
                    {{else}}
                    <div class="project-card">
                    {{end}}
                        <div class="project-card-header">
                            <span class="project-name">{{icon .Icon "project-icon"}} {{.Name}}</span>
                            <span class="project-link-icon">{{if .IsLink}}{{icon "arrow-up-right-from-square" ""}}{{else}}{{icon "server" ""}}{{end}}</span>
                        </div>
                        <p class="project-desc">{{.Description}}</p>
                        <span class="project-url"{{if not .IsLink}} style="cursor: default"{{end}}>{{.URLLabel}}</span>
                    {{if .IsLink}}
                    </a>
                    {{else}}
                    </div>
                    {{end}}
                    {{end}}
                </div>
            </div>
        </section>

        <section id="about">
            <div class="container">
                <p class="section-label">// about</p>
                <h2 class="section-title">About me</h2>
                <div class="about-inner">
                    <img src="/static/avatar.png" alt="{{.Profile.Name}}" class="about-photo" />
                    <div class="about-content">
                        <div class="about-meta">
                            <span>age</span>
                            <span class="meta-val" id="age">-</span>
                            <span>based in</span>
                            <span class="meta-val">{{.Profile.Location}}</span>
                            <span>handle</span>
                            <span class="meta-val">@{{.Profile.Handle}}</span>
                        </div>
                        <p>{{.AboutPara1}}</p>
                        <p>{{.AboutPara2}}</p>

                        <p class="about-subheading" style="margin-top: 1.5rem">// experience</p>
                        <div class="timeline">
                            {{range .Experience}}
                            <div class="tl-row">
                                <span class="tl-years">{{.Years}}</span>
                                <span class="tl-role">{{.Role}} {{icon .Icon "tl-icon"}}</span>
                                <span class="tl-company">{{.Company}}</span>
                            </div>
                            {{end}}
                        </div>

                        <p class="about-subheading">// skills</p>
                        <div class="skills-list">
                            {{range .Skills}}<span class="skill-tag">{{.Name}}</span>{{end}}
                        </div>
                    </div>
                </div>
            </div>
        </section>

        <footer>
            <p>
                johansen.foo &mdash; query
                <a href="/me"><code>GET /me</code></a>
                for JSON &mdash;
                <a href="https://github.com/mojoaar" target="_blank" rel="noopener noreferrer">github.com/mojoaar</a>
            </p>
        </footer>

        <script>
            document.addEventListener("DOMContentLoaded", () => {
                const toggle = document.getElementById("themeToggle");
                toggle.addEventListener("click", () => {
                    const current = document.documentElement.getAttribute("data-mode");
                    const next = current === "light" ? "dark" : "light";
                    document.documentElement.setAttribute("data-mode", next);
                    localStorage.setItem("mode", next);
                });

                const hamburger = document.getElementById("hamburger");
                const navLinks = document.getElementById("navLinks");

                hamburger.addEventListener("click", () => {
                    const open = navLinks.classList.toggle("open");
                    hamburger.setAttribute("aria-expanded", open);
                });

                navLinks.querySelectorAll("a").forEach((link) => {
                    link.addEventListener("click", () => {
                        navLinks.classList.remove("open");
                        hamburger.setAttribute("aria-expanded", "false");
                    });
                });

                const dob = new Date("{{.Profile.DOB}}");
                const now = new Date();
                let age = now.getFullYear() - dob.getFullYear();
                const m = now.getMonth() - dob.getMonth();
                if (m < 0 || (m === 0 && now.getDate() < dob.getDate())) age--;
                document.getElementById("age").textContent = age;

                function equalizeCards() {
                    const cards = document.querySelectorAll(".project-card");
                    cards.forEach((c) => (c.style.height = "auto"));
                    let max = 0;
                    cards.forEach((c) => {
                        max = Math.max(max, c.offsetHeight);
                    });
                    cards.forEach((c) => (c.style.height = max + "px"));
                }
                equalizeCards();
                window.addEventListener("resize", equalizeCards);
            });
        </script>
{{end}}
```

Three deliberate differences from the old markup, all of which preserve appearance:

1. The inline-styled `johansen.foo` code chip becomes `class="inline-code"`. Append this rule to `internal/web/static/style.css` so the style moves out of the markup while staying identical:

```css
/* ─── Inline code chip ──────────────────────────────────────────── */
.inline-code {
    font-family: var(--font-mono);
    font-size: 0.85em;
    background: var(--bg3);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 0.1rem 0.4rem;
}
```

2. `arrow-up-right-from-square` is a real vendored icon now, so the glyph is inline SVG rather than a Font Awesome `<i>`. The `homelab` card keeps `server` as its trailing glyph, exactly as before.

3. The Lucide `<script type="module">` block is gone entirely.

The `.hero-title` in the old page is hardcoded with a leading `//`, and `profile.tagline` already stores `// enterprise it leader & open-source tinkerer`, so no prefix is added in the template.

- [ ] **Step 4: Implement template loading and the renderer**

`internal/web/render.go`:

```go
package web

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/icons"
	"github.com/mojoaar/johansenfoo/internal/markdown"
)

//go:embed templates/*.html
var templateFS embed.FS

type page struct {
	Profile        db.Profile
	Social         []db.SocialLink
	Projects       []db.Project
	Experience     []db.Experience
	Skills         []db.Skill
	HeroBio        template.HTML
	AboutPara1     template.HTML
	AboutPara2     template.HTML
	SiteName       string
	ThemeSlug      string
	ThemeCSS       template.CSS
	Meta           Meta
	StructuredData template.HTML
}

var templates = template.Must(
	template.New("").Funcs(template.FuncMap{
		"icon": func(name, class string) template.HTML {
			return icons.Inline(name, class)
		},
	}).ParseFS(templateFS, "templates/*.html"),
)

func renderPage(w http.ResponseWriter, name string, data page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func newPage(c *db.SiteContent, themeCSS string, meta Meta, structured template.HTML) page {
	return page{
		Profile:        c.Profile,
		Social:         c.Social,
		Projects:       c.Projects,
		Experience:     c.Experience,
		Skills:         c.Skills,
		HeroBio:        markdown.Render(c.Profile.HeroBio),
		AboutPara1:     markdown.Render(c.Profile.AboutPara1),
		AboutPara2:     markdown.Render(c.Profile.AboutPara2),
		ThemeSlug:      c.Theme.Slug,
		ThemeCSS:       template.CSS(themeCSS),
		Meta:           meta,
		StructuredData: structured,
	}
}
```

- [ ] **Step 5: Write the failing landing test**

`internal/web/landing_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLandingRendersContentFromDatabase(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Morten Johansen",
		"// enterprise it leader &amp; open-source tinkerer",
		"atlascmdb",
		"mindmatrix",
		"self-hosted // private",
		"Jydske Dragonregiment",
		"ITSM",
		"Presenting",
		`href="https://jysk.com"`,
		`data-theme="johansen"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("landing page is missing %q", want)
		}
	}

	if strings.Contains(body, "cdn.jsdelivr.net") {
		t.Error("landing page still loads Lucide from a CDN")
	}
	if strings.Contains(body, "font-awesome") {
		t.Error("landing page still loads Font Awesome")
	}
	if strings.Contains(body, "data-lucide=") {
		t.Error("landing page still contains data-lucide placeholders")
	}
}

func TestLandingInlineThemeIsPresent(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "--bg:#0f1117") {
		t.Error("landing page does not inline the active theme")
	}
}
```

The handler must be given a seeded database, so `newTestHandler` needs to build one. Update it in `internal/web/web_test.go`:

```go
func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := db.Migrate(d); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}

	content, err := loadContent(d)
	if err != nil {
		t.Fatalf("loadContent: %v", err)
	}

	return New(Deps{
		DB:      d,
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Content: content,
		Version: "test",
		Started: time.Now(),
	})
}
```

Add `"path/filepath"` and `"github.com/mojoaar/johansenfoo/internal/db"` to the imports, and drop the old single-line body.

- [ ] **Step 6: Run the test to verify it fails**

Run: `go test ./internal/web/ -run TestLanding -v`
Expected: FAIL — `undefined: loadContent`, and `Deps` has no `Content` field.

- [ ] **Step 7: Implement content loading and the landing handler**

`internal/web/landing.go`:

```go
package web

import (
	"database/sql"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

func loadContent(d *sql.DB) (*db.SiteContent, error) {
	profile, err := db.NewProfileRepo(d).Get()
	if err != nil {
		return nil, err
	}
	social, err := db.NewProfileRepo(d).SocialLinks()
	if err != nil {
		return nil, err
	}
	content := db.NewContentRepo(d)
	projects, err := content.Projects()
	if err != nil {
		return nil, err
	}
	experience, err := content.Experience()
	if err != nil {
		return nil, err
	}
	skills, err := content.Skills()
	if err != nil {
		return nil, err
	}

	settings, err := db.NewSettingsRepo(d).All()
	if err != nil {
		return nil, err
	}

	slug := settings["active_theme"]
	row, err := db.NewThemeRepo(d).GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	return &db.SiteContent{
		Profile:    *profile,
		Social:     social,
		Projects:   projects,
		Experience: experience,
		Skills:     skills,
		Theme:      *row,
		Settings:   settings,
	}, nil
}

func themeFromRow(row db.Theme) theme.Theme {
	return theme.Theme{
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		Base:        row.TokensBase,
		Light:       row.TokensLight,
		Dark:        row.TokensDark,
	}
}

func landingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content
		css := theme.CSS(themeFromRow(c.Theme))
		meta := resolveMeta(c, "/")
		data := newPage(c, css, meta, personSchema(c))
		renderPage(w, "base", data)
	}
}
```

`db.SiteContent.Theme` stays typed `db.Theme` so the `db` package never imports `theme`; `themeFromRow` bridges the two at the render boundary.

Add `All()` to `internal/db/repo_settings.go`:

```go
func (r *SettingsRepo) All() (map[string]string, error) {
	rows, err := r.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}
```

Add `Content *db.SiteContent` to `Deps` in `internal/web/web.go`, and register the route:

```go
	r.Get("/", landingHandler(d))
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/web/ -run TestLanding -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat: render the landing page from the database"
```

---

### Task 9: `/me`, robots.txt, and sitemap.xml

**Files:**
- Create: `internal/web/me.go`, `internal/web/seo.go`
- Create: `internal/web/seo_test.go`

**Interfaces:**
- Consumes: `db.SiteContent`.
- Produces: `Meta` struct; `resolveMeta(c *db.SiteContent, route string) Meta`; `personSchema(c *db.SiteContent) template.HTML`; `meHandler`, `robotsHandler`, `sitemapHandler`.

- [ ] **Step 1: Write the failing tests**

`internal/web/seo_test.go`:

```go
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMeReturnsJSONFromDatabase(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}

	var got struct {
		Name     string   `json:"name"`
		Handle   string   `json:"handle"`
		Location string   `json:"location"`
		DOB      string   `json:"dob"`
		Skills   []string `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if got.Name != "Morten Johansen" || got.Handle != "mojoaar" || got.Location != "Denmark" {
		t.Errorf("identity fields wrong: %+v", got)
	}
	if got.DOB != "1980-08-13" {
		t.Errorf("dob = %q", got.DOB)
	}
	if len(got.Skills) != 30 {
		t.Errorf("got %d skills, want 30", len(got.Skills))
	}
	if len(got.Projects) != 9 {
		t.Errorf("got %d projects, want 9", len(got.Projects))
	}
	if got.Projects[3].Name != "homelab" || got.Projects[3].URL != "" {
		t.Errorf("homelab project = %+v, want empty url", got.Projects[3])
	}
	if got.Social["github"] != "https://github.com/mojoaar" {
		t.Errorf("social github = %q", got.Social["github"])
	}
}

func TestRobotsTxt(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "User-agent: *") {
		t.Error("robots.txt is missing the user-agent line")
	}
	if !strings.Contains(body, "Sitemap: https://johansen.foo/sitemap.xml") {
		t.Error("robots.txt is missing the sitemap line")
	}
}

func TestSitemapXML(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, `<?xml`) {
		t.Error("sitemap does not start with an XML declaration")
	}
	if !strings.Contains(body, "<loc>https://johansen.foo/</loc>") {
		t.Error("sitemap is missing the home URL")
	}
	if strings.Contains(body, "/posts") {
		t.Error("phase 1 has no posts and the sitemap must not advertise them")
	}
}

func TestMetaPrefersSettingsOverFallback(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "<title>Morten Johansen | johansen.foo</title>") {
		t.Error("title does not come from settings")
	}
	if !strings.Contains(body, `<link rel="canonical" href="https://johansen.foo/" />`) {
		t.Error("canonical URL is wrong")
	}
	if !strings.Contains(body, `content="index, follow"`) {
		t.Error("robots meta is wrong")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/web/ -run 'TestMe|TestRobots|TestSitemap|TestMeta' -v`
Expected: FAIL — 404s, and `undefined: resolveMeta`.

- [ ] **Step 3: Implement the SEO resolver**

`internal/web/seo.go`:

```go
package web

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type Meta struct {
	Title       string
	Description string
	Canonical   string
	Robots      string
	OGType      string
	OGImage     string
	TwitterCard string
	ThemeColor  string
}

const defaultDescription = "Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years."

func resolveMeta(c *db.SiteContent, route string) Meta {
	s := c.Settings

	title := s["site_title"]
	if title == "" {
		title = "Morten Johansen | johansen.foo"
	}
	description := s["seo_description"]
	if description == "" {
		description = defaultDescription
	}
	base := s["canonical_base_url"]
	if base == "" {
		base = "https://johansen.foo"
	}
	robots := "index, follow"
	if s["noindex"] == "true" {
		robots = "noindex, nofollow"
	}

	canonical := base + "/"
	if route != "/" {
		canonical = base + route
	}

	ogType := s["og_type"]
	if ogType == "" {
		ogType = "website"
	}
	card := s["twitter_card"]
	if card == "" {
		card = "summary"
	}

	return Meta{
		Title:       title,
		Description: description,
		Canonical:   canonical,
		Robots:      robots,
		OGType:      ogType,
		OGImage:     s["og_image_url"],
		TwitterCard: card,
		ThemeColor:  themeColor(c),
	}
}

func themeColor(c *db.SiteContent) string {
	if v := c.Theme.Dark["--bg"]; v != "" {
		return v
	}
	return "#0f1117"
}

func personSchema(c *db.SiteContent) template.HTML {
	sameAs := make([]string, 0, len(c.Social))
	for _, s := range c.Social {
		if s.Platform != "github" {
			sameAs = append(sameAs, s.URL)
		}
	}
	for _, s := range c.Social {
		if s.Platform == "github" {
			sameAs = append(sameAs, s.URL)
		}
	}

	knowsAbout := make([]string, 0, len(c.Skills))
	for _, s := range c.Skills {
		knowsAbout = append(knowsAbout, s.Name)
	}

	payload := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "Person",
		"name":        c.Profile.Name,
		"url":         c.Settings["canonical_base_url"],
		"image":       c.Settings["og_image_url"],
		"jobTitle":    "Enterprise IT Leader",
		"description": c.Settings["seo_description"],
		"sameAs":      sameAs,
		"knowsAbout":  knowsAbout,
	}

	body, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(body) + `</script>`)
}
```

The `sameAs` ordering matches the current page, which lists bluesky, linkedin, mastodon, github. Because `social_link.sort` already gives that order and github is last, the two loops above reproduce it without hardcoding.

`json.MarshalIndent` escapes `<`, `>`, and `&` inside strings, which is exactly what we want inside a `<script>` block — no `</script>` can be injected. `template.HTML` is correct for this generated payload.

- [ ] **Step 4: Implement the `/me` handler**

`internal/web/me.go`:

```go
package web

import (
	"encoding/json"
	"net/http"
)

type meProject struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type meResponse struct {
	Name     string            `json:"name"`
	Handle   string            `json:"handle"`
	Location string            `json:"location"`
	DOB      string            `json:"dob"`
	Bio      string            `json:"bio"`
	Skills   []string          `json:"skills"`
	Social   map[string]string `json:"social"`
	Projects []meProject       `json:"projects"`
}

func meHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content

		skills := make([]string, 0, len(c.Skills))
		for _, s := range c.Skills {
			skills = append(skills, s.Name)
		}

		projects := make([]meProject, 0, len(c.Projects))
		for _, p := range c.Projects {
			projects = append(projects, meProject{
				Name:        p.Name,
				URL:         p.URL,
				Description: p.Description,
			})
		}

		social := make(map[string]string, len(c.Social))
		for _, s := range c.Social {
			social[s.Platform] = s.URL
		}

		payload, err := json.MarshalIndent(meResponse{
			Name:     c.Profile.Name,
			Handle:   c.Profile.Handle,
			Location: c.Profile.Location,
			DOB:      c.Profile.DOB,
			Bio:      c.Profile.HeroBio,
			Skills:   skills,
			Social:   social,
			Projects: projects,
		}, "", "  ")
		if err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(append(payload, '\n'))
	}
}
```

Encoding into a buffer before touching the response means a marshal failure can still produce a clean 500 — calling `http.Error` after the encoder has already written a 200 would only emit a superfluous-header warning. `meResponse` is built entirely from strings and slices, so `MarshalIndent` cannot actually fail here; the guard exists so the failure mode is correct rather than merely unreachable.

`Bio` uses `HeroBio`, which is what the old `me` file contained as `bio`. Note that `HeroBio` is now markdown and therefore contains a `[JYSK](https://jysk.com)` link, whereas the old `me` had plain prose. If exact `/me` parity matters, store the API bio separately — the pragmatic call is to accept the markdown form, since `/me` is a machine-readable endpoint and the link is strictly more useful. Confirm with the user when reviewing this phase.

- [ ] **Step 5: Implement robots.txt and sitemap.xml**

Append to `internal/web/seo.go`:

```go
func robotsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body := d.Content.Settings["robots_txt"]
		if body == "" {
			body = "User-agent: *\nAllow: /\n"
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func sitemapHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base := d.Content.Settings["canonical_base_url"]
		if base == "" {
			base = "https://johansen.foo"
		}

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
		w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n"))
		w.Write([]byte("  <url>\n"))
		w.Write([]byte("    <loc>" + base + "/</loc>\n"))
		w.Write([]byte("  </url>\n"))
		w.Write([]byte("</urlset>\n"))
	}
}
```

Phase 4 adds the posts and tag URLs here, gated on `posts_enabled`.

- [ ] **Step 6: Register the routes**

In `internal/web/web.go`, after `r.Get("/", landingHandler(d))`:

```go
	r.Get("/me", meHandler(d))
	r.Get("/robots.txt", robotsHandler(d))
	r.Get("/sitemap.xml", sitemapHandler(d))
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `go test ./internal/web/ -v`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "feat: add /me, robots.txt, sitemap.xml, and SEO metadata resolution"
```

---

### Task 10: Wire the real database into main and verify parity

**Files:**
- Modify: `cmd/johansenfoo/main.go`
- Create: `cmd/johansenfoo/main_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces: a bootable binary serving the whole Phase 1 surface.

- [ ] **Step 1: Wire the database into `main`**

In `cmd/johansenfoo/main.go`, replace the `config.Load` / `web.New` block with:

```go
	cfg, err := config.Load(dataDir)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	if err := db.Migrate(d); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	content, err := web.LoadContent(d)
	if err != nil {
		return fmt.Errorf("load content: %w", err)
	}

	handler := web.New(web.Deps{
		DB:      d,
		Cfg:     cfg,
		Content: content,
		Version: version,
		Started: time.Now(),
	})
```

Add `"github.com/mojoaar/johansenfoo/internal/db"` to the imports.

This requires `loadContent` to be exported. Rename it to `LoadContent` in `internal/web/landing.go` and update the call sites in `internal/web/web_test.go` and `internal/web/landing.go`.

- [ ] **Step 2: Write the boot test**

`cmd/johansenfoo/main_test.go`:

```go
package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunBootsAndServes(t *testing.T) {
	h, _, cleanup, err := buildHandler(t.TempDir())
	if err != nil {
		t.Fatalf("buildHandler: %v", err)
	}
	defer cleanup()

	cases := []struct {
		path string
		want int
	}{
		{"/", http.StatusOK},
		{"/me", http.StatusOK},
		{"/health", http.StatusOK},
		{"/robots.txt", http.StatusOK},
		{"/sitemap.xml", http.StatusOK},
		{"/static/style.css", http.StatusOK},
		{"/nope", http.StatusNotFound},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("GET %s = %d, want %d", c.path, rec.Code, c.want)
		}
	}
}
```

- [ ] **Step 3: Extract `buildHandler`**

Refactor `run` so the wiring is testable. In `cmd/johansenfoo/main.go`:

```go
func buildHandler(dataDir string) (http.Handler, *config.Config, func(), error) {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	d, err := db.Open(cfg.DBPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Migrate(d); err != nil {
		_ = d.Close()
		return nil, nil, nil, fmt.Errorf("migrate: %w", err)
	}

	content, err := web.LoadContent(d)
	if err != nil {
		_ = d.Close()
		return nil, nil, nil, fmt.Errorf("load content: %w", err)
	}

	handler := web.New(web.Deps{
		DB:      d,
		Cfg:     cfg,
		Content: content,
		Version: version,
		Started: time.Now(),
	})
	return handler, cfg, func() { _ = d.Close() }, nil
}

func run(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}

	handler, cfg, cleanup, err := buildHandler(dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	// ...unchanged signal handling...
}
```

`buildHandler` returns the config it already loaded, so `run` never parses `config.json` twice.

- [ ] **Step 4: Run the full suite and the vet pass**

```bash
cd /Users/mojoaar/Development/johansenfoo
go vet ./... && go test ./...
```

Expected: all PASS.

- [ ] **Step 5: Verify parity by hand against the live site**

```bash
go build -o /tmp/johansenfoo ./cmd/johansenfoo
rm -rf /tmp/jf-data && /tmp/johansenfoo -data=/tmp/jf-data &
sleep 1

echo "=== our headers ==="
curl -sS -o /dev/null -D - http://localhost:8080/ | head -8
echo "=== live site headers ==="
curl -sS -o /dev/null -D - https://johansen.foo/ | head -8

echo "=== our /me content-type ==="
curl -sS -o /dev/null -D - http://localhost:8080/me | grep -i content-type
echo "(live incorrectly omits application/json; ours must include it)"

echo "=== structural comparison ==="
curl -sS http://localhost:8080/ > /tmp/ours.html
grep -oE 'class="[a-z-]+"' /tmp/ours.html | sort -u > /tmp/ours-classes.txt
curl -sS https://johansen.foo/ > /tmp/theirs.html
grep -oE 'class="[a-z-]+"' /tmp/theirs.html | sort -u > /tmp/theirs-classes.txt
diff /tmp/theirs-classes.txt /tmp/ours-classes.txt

kill %1
```

Expected: the class lists differ only by `inline-code` (ours) and the Font Awesome classes the old page no longer needs. Every content string is present. `/me` returns `application/json; charset=utf-8`.

- [ ] **Step 6: Verify the counts against the old `me` file**

```bash
python3 - <<'PY'
import json, urllib.request
old = json.load(open('/Users/mojoaar/Development/johansen_landing/me'))
new = json.load(urllib.request.urlopen('http://localhost:8080/me'))

for field in ('name', 'handle', 'location', 'dob'):
    assert old[field] == new[field], (field, old[field], new[field])
assert old['skills'] == new['skills'], 'skills differ'
assert old['social'] == new['social'], 'social differs'
assert [p['name'] for p in old['projects']] == [p['name'] for p in new['projects']], 'project names differ'
assert [p['url'] for p in old['projects']] == [p['url'] for p in new['projects']], 'project urls differ'
assert [p['description'] for p in old['projects']] == [p['description'] for p in new['projects']], 'descriptions differ'
print('PARITY OK: identity, skills, social, projects, and descriptions all match')
PY
```

Run this with the server still up. Expected: `PARITY OK`.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "feat: wire database and content loading into the binary"
```

---

## Phase 1 Done When

- `go test ./...` and `go vet ./...` pass.
- `GET /` renders the landing page from SQLite with no third-party origins.
- `GET /me` returns the same identity, skills, social links and projects as the current static `me` file, with `Content-Type: application/json; charset=utf-8`.
- `GET /robots.txt` and `GET /sitemap.xml` are generated.
- `GET /health` returns `ok`, and `GET /static/*` serves the ported stylesheet, fonts, favicons and avatar.
- The only difference from the current site's markup is the removal of the Lucide runtime and the Font Awesome webfont, both replaced by inline SVG.
