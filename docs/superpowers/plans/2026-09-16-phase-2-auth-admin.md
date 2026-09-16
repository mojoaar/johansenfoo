# Phase 2 — Auth & Admin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the site an owner-only admin surface — bcrypt password, session login, CSRF, and an HTMX CRUD UI for profile, social links, projects, experience and skills — where every write is immediately visible on the public site.

**Architecture:** All Phase 2 changes live inside `internal/` plus `cmd/johansenfoo/main.go`. A new `session` table backs cookie logins. The Phase 1 `Deps.Content *db.SiteContent` startup snapshot is replaced by a `*ContentStore` holding an `atomic.Pointer[db.SiteContent]` that is reloaded after every write — this is the carried-forward gate that makes "content changes take effect on the next request" true. Admin HTML is server-rendered with `html/template` and driven by vendored HTMX; there is no JavaScript build step and no framework.

**Tech Stack:** Go 1.25, `github.com/go-chi/chi/v5`, `modernc.org/sqlite` (CGO-free), `html/template`, `golang.org/x/crypto/bcrypt`, vendored `htmx.org` 2.0.4.

## Global Constraints

Copied verbatim from the approved spec and the Phase 1 plan. Every task's requirements implicitly include this section.

- Module path: `github.com/mojoaar/johansenfoo`. Go directive: `go 1.25.5`.
- No CGO anywhere: every command runs as `CGO_ENABLED=0 go test ./...` / `CGO_ENABLED=0 go build ./...`.
- No comments in Go code unless essential. `//go:embed` is the only sanctioned exception.
- No third-party origin on the runtime critical path. The single exception is the existing `umami.johansen.foo` analytics snippet in `base.html`. HTMX must be vendored into `internal/web/static/`, never loaded from a CDN.
- No JavaScript framework. HTMX only.
- All timestamps are stored and emitted as RFC 3339 UTC. In SQL that means `strftime('%Y-%m-%dT%H:%M:%SZ','now')` — never `datetime('now')`. In Go that means `time.Now().UTC().Format(time.RFC3339)`.
- `data-theme` on `<html>` is the theme SLUG; `data-mode` is `light` or `dark`. They are separate axes and must stay separate.
- Structured text content is markdown rendered with goldmark via `internal/markdown`.
- Every task ends with `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=0 go vet ./...` passing, then a commit.
- Migrations are append-only. `0001`–`0004` must not be edited. The next migration number is `0005`.
- No changes to `internal/theme` token emission in this phase. Phase 2 introduces no theme write path; `theme.Validate` still only rejects `{`, `}` and `;`, which is not sufficient for untrusted input.

## Carried-forward gate this phase must close

The Phase 1 whole-branch review recorded this as a hard gate:

> `Deps.Content` is a startup snapshot with NO reload/invalidation path — it contradicts the spec's promise that content changes take effect on the next request.

Task 2 closes it. Tasks 6–9 depend on it.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/db/migrations/0005_session.sql` | Create the `session` table and its expiry index |
| `internal/db/repo_session.go` | `SessionRepo`: create, look up, delete, prune expired sessions |
| `internal/db/repo_settings.go` | Add `Set(key, value string) error` |
| `internal/db/repo_profile.go` | Add `Update(*Profile) error` and social-link create/update/delete |
| `internal/db/repo_content.go` | Add project/experience/skill create, update, delete, and reorder |
| `internal/web/content.go` | `ContentStore` — atomic snapshot with `Current()` / `Reload()` |
| `internal/web/auth.go` | Password storage, bcrypt, setup/login/logout, sessions, rate limiting, `authMiddleware` |
| `internal/web/csrf.go` | `csrfMiddleware`, `methodOverride`, CSRF cookie issuance |
| `internal/web/admin.go` | Admin route table, dashboard, shared page helper |
| `internal/web/admin_profile.go` | Profile + social-link admin handlers |
| `internal/web/admin_projects.go` | Project admin handlers |
| `internal/web/admin_experience.go` | Experience admin handlers |
| `internal/web/admin_skills.go` | Skill admin handlers |
| `internal/web/admin_security.go` | Password change and API-key generate/regenerate |
| `internal/web/templates/admin_layout.html` | `admin_head` / `admin_sidebar` partials |
| `internal/web/templates/admin_dashboard.html` | Dashboard page |
| `internal/web/templates/admin_profile.html` | Profile + social links page |
| `internal/web/templates/admin_projects.html` | Project list page |
| `internal/web/templates/admin_project_form.html` | Project create/edit form |
| `internal/web/templates/admin_experience.html` | Experience list page |
| `internal/web/templates/admin_experience_form.html` | Experience create/edit form |
| `internal/web/templates/admin_skills.html` | Skill list + inline add page |
| `internal/web/templates/admin_security.html` | Password + API key page |
| `internal/web/templates/admin_login.html` | Login page (standalone) |
| `internal/web/templates/admin_setup.html` | First-run setup page (standalone) |
| `internal/web/static/admin.css` | Admin stylesheet |
| `internal/web/static/htmx.min.js` | Vendored HTMX 2.0.4 |
| `internal/web/web.go` | Modified: `Deps.Content` type, admin routes, middleware order |
| `internal/web/render.go` | Modified: `page` gains admin fields; `newAdminPage` helper |
| `internal/web/landing.go` | Modified: handlers read `d.Content.Current()` |
| `cmd/johansenfoo/main.go` | Modified: build the `ContentStore` instead of a one-shot snapshot |

---

### Task 1: Session storage

**Files:**
- Create: `internal/db/migrations/0005_session.sql`
- Create: `internal/db/repo_session.go`
- Test: `internal/db/repo_session_test.go`

**Interfaces:**
- Consumes: the existing `seeded(t *testing.T) *sql.DB` helper from `internal/db/seed_test.go` (package-private, same package).
- Produces: `db.Session{ID string; CreatedAt, ExpiresAt time.Time}`, `db.ErrSessionNotFound`, `db.NewSessionRepo(d *sql.DB) *SessionRepo`, `(*SessionRepo).Create(id string, expiresAt time.Time) error`, `(*SessionRepo).Get(id string) (*Session, error)`, `(*SessionRepo).Delete(id string) error`, `(*SessionRepo).DeleteExpired() (int64, error)`.

- [ ] **Step 1: Write the failing test**

Create `internal/db/repo_session_test.go`:

```go
package db

import (
	"errors"
	"testing"
	"time"
)

func TestSessionCreateAndGet(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	expires := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	if err := r.Create("abc123", expires); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s, err := r.Get("abc123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.ID != "abc123" {
		t.Errorf("ID = %q, want %q", s.ID, "abc123")
	}
	if !s.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", s.ExpiresAt, expires)
	}
	if s.ExpiresAt.Location() != time.UTC {
		t.Errorf("ExpiresAt location = %v, want UTC", s.ExpiresAt.Location())
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestSessionGetMissing(t *testing.T) {
	d := seeded(t)
	if _, err := NewSessionRepo(d).Get("nope"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionDelete(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	if err := r.Create("gone", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Delete("gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.Get("gone"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("after Delete err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionDeleteExpired(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := r.Create("old", past); err != nil {
		t.Fatalf("Create old: %v", err)
	}
	if err := r.Create("live", future); err != nil {
		t.Fatalf("Create live: %v", err)
	}

	n, err := r.DeleteExpired()
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted = %d, want 1", n)
	}
	if _, err := r.Get("live"); err != nil {
		t.Errorf("live session should survive: %v", err)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestSession -v`
Expected: FAIL — the `internal/db` package does not compile because `NewSessionRepo`, `ErrSessionNotFound` and the `session` table do not exist.

- [ ] **Step 3: Write the migration**

Create `internal/db/migrations/0005_session.sql`:

```sql
CREATE TABLE IF NOT EXISTS session (
    id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_expires_at ON session (expires_at);
```

- [ ] **Step 4: Write the repository**

Create `internal/db/repo_session.go`:

```go
package db

import (
	"database/sql"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type SessionRepo struct{ db *sql.DB }

func NewSessionRepo(d *sql.DB) *SessionRepo { return &SessionRepo{db: d} }

func (r *SessionRepo) Create(id string, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO session (id, created_at, expires_at)
		 VALUES (?, strftime('%Y-%m-%dT%H:%M:%SZ','now'), ?)`,
		id, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (r *SessionRepo) Get(id string) (*Session, error) {
	var s Session
	var created, expires string
	err := r.db.QueryRow(
		`SELECT id, created_at, expires_at FROM session WHERE id = ?`, id,
	).Scan(&s.ID, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.CreatedAt, err = time.Parse(time.RFC3339, created); err != nil {
		return nil, err
	}
	if s.ExpiresAt, err = time.Parse(time.RFC3339, expires); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SessionRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM session WHERE id = ?`, id)
	return err
}

func (r *SessionRepo) DeleteExpired() (int64, error) {
	res, err := r.db.Exec(
		`DELETE FROM session WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

The lexicographic comparison in `DeleteExpired` is correct because every timestamp is fixed-width RFC 3339 UTC with a `Z` suffix.

- [ ] **Step 5: Run the test to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestSession -v`
Expected: PASS — all four tests green.

- [ ] **Step 6: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent. The existing `TestSchemaHasCoreTables` and `TestMigrateIsIdempotent` still pass — `Migrate` picks up `0005` automatically from the embedded FS.

- [ ] **Step 7: Commit**

```bash
git add internal/db/migrations/0005_session.sql internal/db/repo_session.go internal/db/repo_session_test.go
git commit -m "feat: add session table and repository"
```

---

### Task 2: Writable settings and the content store

This is the carried-forward gate. After this task the public site re-renders from the database on the next request after any write, instead of serving a startup snapshot forever.

**Files:**
- Modify: `internal/db/repo_settings.go`
- Create: `internal/web/content.go`
- Modify: `internal/web/web.go` (`Deps.Content` type)
- Modify: `internal/web/landing.go` (handlers use `.Current()`)
- Modify: `internal/web/render.go` (unchanged signatures)
- Modify: `internal/web/web_test.go`
- Modify: `internal/web/landing_test.go`
- Modify: `internal/web/seo_test.go`
- Modify: `cmd/johansenfoo/main.go`
- Test: `internal/web/content_test.go`

**Interfaces:**
- Consumes: `db.NewSettingsRepo`, `db.ErrSettingNotFound`, `web.LoadContent(d *sql.DB) (*db.SiteContent, error)`.
- Produces: `(*db.SettingsRepo).Set(key, value string) error`; `web.ContentStore` with `web.NewContentStore(d *sql.DB) (*ContentStore, error)`, `(*ContentStore).Current() *db.SiteContent`, `(*ContentStore).Reload() error`; `web.Deps.Content` becomes `*ContentStore` instead of `*db.SiteContent`.

- [ ] **Step 1: Write the failing settings test**

Append to `internal/db/repo_test.go`:

```go
func TestSettingsRepoSet(t *testing.T) {
	d := seeded(t)
	r := NewSettingsRepo(d)

	if err := r.Set("site_title", "New Title"); err != nil {
		t.Fatalf("Set existing: %v", err)
	}
	got, err := r.Get("site_title")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "New Title" {
		t.Errorf("site_title = %q, want %q", got, "New Title")
	}

	if err := r.Set("brand_new_key", "hello"); err != nil {
		t.Fatalf("Set new: %v", err)
	}
	if got, err = r.Get("brand_new_key"); err != nil || got != "hello" {
		t.Errorf("brand_new_key = %q, %v; want %q, nil", got, err, "hello")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestSettingsRepoSet -v`
Expected: FAIL with `r.Set undefined (type *SettingsRepo has no field or method Set)`.

- [ ] **Step 3: Implement `Set`**

Append to `internal/db/repo_settings.go`:

```go
func (r *SettingsRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
```

- [ ] **Step 4: Run it to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run TestSettingsRepoSet -v`
Expected: PASS.

- [ ] **Step 5: Write the failing content-store test**

Create `internal/web/content_test.go`:

```go
package web

import (
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestContentStoreCurrentReturnsSnapshot(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	c := s.Current()
	if c == nil {
		t.Fatal("Current() returned nil")
	}
	if c.Profile.Name != "Morten Johansen" {
		t.Errorf("Profile.Name = %q", c.Profile.Name)
	}
	if len(c.Projects) != 9 {
		t.Errorf("len(Projects) = %d, want 9", len(c.Projects))
	}
}

func TestContentStoreReloadPicksUpWrites(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	if s.Current().Profile.Name == "Changed Name" {
		t.Fatal("precondition failed: name already changed")
	}

	if _, err := d.Exec(`UPDATE profile SET name = 'Changed Name' WHERE id = 1`); err != nil {
		t.Fatalf("update: %v", err)
	}
	if s.Current().Profile.Name == "Changed Name" {
		t.Fatal("Current() reflected the write before Reload()")
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := s.Current().Profile.Name; got != "Changed Name" {
		t.Errorf("after Reload Profile.Name = %q, want %q", got, "Changed Name")
	}
}

func TestContentStoreReloadErrorLeavesPreviousSnapshot(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	before := s.Current()

	if _, err := d.Exec(`DELETE FROM profile`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.Reload(); err == nil {
		t.Fatal("Reload() succeeded with no profile row, want error")
	}
	if s.Current() != before {
		t.Error("failed Reload() replaced the snapshot")
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestContentStore -v`
Expected: FAIL — `undefined: NewContentStore` and `undefined: newTestDB`.

- [ ] **Step 7: Add the shared test database helper**

The Phase 1 `internal/web/web_test.go` inline-seeds a database. Replace that duplication with one helper. Open `internal/web/web_test.go` and replace its `TestHealth` preamble block that reads `d := ...` with a call to `newTestDB(t)`, then append this helper at the end of the file:

```go
func newTestDB(t *testing.T) *sql.DB {
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
	return d
}
```

Ensure `internal/web/web_test.go` imports `database/sql`, `path/filepath`, `github.com/mojoaar/johansenfoo/internal/db` and `testing`.

- [ ] **Step 8: Implement the content store**

Create `internal/web/content.go`:

```go
package web

import (
	"database/sql"
	"sync/atomic"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type ContentStore struct {
	db  *sql.DB
	cur atomic.Pointer[db.SiteContent]
}

func NewContentStore(d *sql.DB) (*ContentStore, error) {
	s := &ContentStore{db: d}
	if err := s.Reload(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *ContentStore) Current() *db.SiteContent {
	return s.cur.Load()
}

func (s *ContentStore) Reload() error {
	c, err := LoadContent(s.db)
	if err != nil {
		return err
	}
	s.cur.Store(c)
	return nil
}
```

- [ ] **Step 9: Run it to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestContentStore -v`
Expected: PASS for all three tests.

- [ ] **Step 10: Switch `Deps.Content` to the store**

In `internal/web/web.go`, change the `Deps` field:

```go
type Deps struct {
	DB      *sql.DB
	Cfg     *config.Config
	Content *ContentStore
	Version string
	Started time.Time
}
```

In `internal/web/landing.go`, in `landingHandler`, change the snapshot read:

```go
func landingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		themeCSS := theme.CSS(themeFromRow(c.Theme))
		meta := resolveMeta(c, "/")
		data := newPage(c, themeCSS, meta, personSchema(c))
		renderPage(w, "base", data)
	}
}
```

In `internal/web/me.go`, do the same at the top of `meHandler`: replace the direct `d.Content` field access with
`c := d.Content.Current()` plus the same nil guard.

In `internal/web/seo.go`, do the same at the top of `robotsHandler` and `sitemapHandler`.

- [ ] **Step 11: Update the tests that construct `Deps`**

In `internal/web/web_test.go`, `internal/web/landing_test.go` and `internal/web/seo_test.go`, replace every occurrence of

```go
content, err := LoadContent(d)
```

with

```go
content, err := NewContentStore(d)
```

and every `Deps{... Content: <*db.SiteContent> ...}` construction keeps the same field name but now passes the store. Where a test builds `Deps` without a DB, it must now pass `Content: store`.

- [ ] **Step 12: Update `main.go`**

In `cmd/johansenfoo/main.go`, inside `buildHandler`, replace

```go
content, err := web.LoadContent(d)
```

with

```go
store, err := web.NewContentStore(d)
```

and pass `Content: store` in `web.New(web.Deps{...})`.

- [ ] **Step 13: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 14: Commit**

```bash
git add internal/db/repo_settings.go internal/db/repo_test.go internal/web/content.go internal/web/content_test.go internal/web/web.go internal/web/web_test.go internal/web/landing.go internal/web/landing_test.go internal/web/me.go internal/web/seo.go internal/web/seo_test.go cmd/johansenfoo/main.go
git commit -m "feat: reload site content from the database instead of a startup snapshot"
```

---

### Task 3: Password, setup, login and logout

**Files:**
- Create: `internal/web/auth.go`
- Create: `internal/web/templates/admin_login.html`
- Create: `internal/web/templates/admin_setup.html`
- Test: `internal/web/auth_test.go`
- Modify: `internal/web/render.go` (extend the `page` struct)
- Modify: `internal/web/web.go` (routes + `authMiddleware`)

**Interfaces:**
- Consumes: `db.NewSettingsRepo`, `db.NewSessionRepo`, `db.ErrSettingNotFound`, `ContentStore`.
- Produces: `web.sessionCookieName = "johansenfoo_session"`, `web.csrfCookieName = "johansenfoo_csrf"`, `web.passwordHashKey = "admin_password_hash"`, `web.randomToken() string`, `web.secureRequest(r *http.Request) bool`, `web.currentSession(d Deps, r *http.Request) bool`, `web.startSession(d Deps, w http.ResponseWriter, r *http.Request) error`, `web.authMiddleware(d Deps) func(http.Handler) http.Handler`, `web.loginHandler(d Deps) http.HandlerFunc`, `web.setupHandler(d Deps) http.HandlerFunc`, `web.logoutHandler(d Deps) http.HandlerFunc`, `web.NewAdminPage(d Deps, r *http.Request, section, title string) page`.

- [ ] **Step 1: Write the failing test**

Create `internal/web/auth_test.go`:

```go
package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func passwordConfigured(t *testing.T, d *sql.DB) bool {
	t.Helper()
	_, err := db.NewSettingsRepo(d).Get(passwordHashKey)
	return err == nil
}

func TestSetupCreatesPasswordAndSession(t *testing.T) {
	d := newTestDB(t)
	store, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	h := newAuthTestHandler(t, d, store)

	if passwordConfigured(t, d) {
		t.Fatal("precondition failed: password already configured")
	}

	form := url.Values{"password": {"correct horse battery staple"}}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if !passwordConfigured(t, d) {
		t.Fatal("password was not stored")
	}
	hash, err := db.NewSettingsRepo(d).Get(passwordHashKey)
	if err != nil {
		t.Fatalf("Get hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse battery staple")); err != nil {
		t.Errorf("stored hash does not verify: %v", err)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Error("no session cookie was set")
	}
}

func TestSetupRejectsShortPassword(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"password": {"short"}}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if passwordConfigured(t, d) {
		t.Error("a short password was accepted")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	d := newTestDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("right-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(hash)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			t.Error("a session cookie was issued for a failed login")
		}
	}
}

func TestLoginSucceedsAndSeedsSession(t *testing.T) {
	d := newTestDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("right-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(hash)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"password": {"right-password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	var sid string
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			sid = c.Value
		}
	}
	if sid == "" {
		t.Fatal("no session cookie issued")
	}
	if _, err := db.NewSessionRepo(d).Get(sid); err != nil {
		t.Errorf("session %q not in the database: %v", sid, err)
	}
}

func TestAdminRequiresSession(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

func TestLogoutClearsSession(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	if err := db.NewSessionRepo(d).Create("sess-1", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sess-1"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if _, err := db.NewSessionRepo(d).Get("sess-1"); err == nil {
		t.Error("session row survived logout")
	}
}
```

Add `newAuthTestHandler` to the same file, and give `auth_test.go` this import list: `database/sql`, `net/http`, `net/http/httptest`, `net/url`, `strings`, `testing`, `time`, `github.com/mojoaar/johansenfoo/internal/config`, `github.com/mojoaar/johansenfoo/internal/db`, `golang.org/x/crypto/bcrypt`.

```go
func newAuthTestHandler(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	cfg := &config.Config{Port: 8080, DBPath: "test.db", BaseURL: "https://johansen.foo"}
	return New(Deps{DB: d, Cfg: cfg, Content: store, Version: "test", Started: time.Now()})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestSetup|TestLogin|TestAdminRequires|TestLogout' -v`
Expected: FAIL — the package does not compile; `passwordHashKey`, `sessionCookieName`, `newAuthTestHandler` and the routes do not exist.

- [ ] **Step 3: Extend the `page` struct**

In `internal/web/render.go`, add fields to `page` (keeping every existing field):

```go
	Page      string
	Section   string
	Title     string
	CSRF      string
	Flash     string
	Error     string
	Version   string
	APIKey    string
	IsNew     bool
	Project   db.Project
	Item      db.Experience
	Skill     db.Skill
```

Then append these helpers at the end of `render.go`:

```go
func NewAdminPage(d Deps, r *http.Request, section, title string) page {
	p := page{
		Page:    section,
		Section: section,
		Title:   title,
		Version: d.Version,
	}
	if c := d.Content.Current(); c != nil {
		p.ThemeSlug = c.Theme.Slug
	}
	return p
}

func renderAdmin(w http.ResponseWriter, r *http.Request, name string, data page) {
	data.CSRF = ensureCSRFCookie(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
```

`NewAdminPage` reads the theme slug from the live snapshot, so callers never need to set it themselves.

Create `internal/web/csrf.go` in this task with `ensureCSRFCookie` plus a no-op `csrfMiddleware` (Task 4 replaces the no-op):

```go
package web

import "net/http"

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	token := randomToken()
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   secureRequest(r),
		})
	}
	return token
}

func csrfMiddleware(next http.Handler) http.Handler {
	return next
}
```

Task 4 replaces the no-op `csrfMiddleware` with the real implementation and adds `methodOverride` to this same file.

- [ ] **Step 4: Implement `auth.go`**

Create `internal/web/auth.go`:

```go
package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "johansenfoo_session"
	csrfCookieName    = "johansenfoo_csrf"
	passwordHashKey   = "admin_password_hash"
	sessionLifetime   = 30 * 24 * time.Hour
	minPasswordLength = 12
	loginMaxAttempts  = 10
	loginWindow       = 15 * time.Minute
)

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func passwordHash(d Deps) (string, bool, error) {
	v, err := db.NewSettingsRepo(d.DB).Get(passwordHashKey)
	if errors.Is(err, db.ErrSettingNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, v != "", nil
}

func startSession(d Deps, w http.ResponseWriter, r *http.Request) error {
	id := randomToken()
	expires := time.Now().UTC().Add(sessionLifetime)
	if err := db.NewSessionRepo(d.DB).Create(id, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureRequest(r),
		Expires:  expires,
	})
	return nil
}

func currentSession(d Deps, r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	s, err := db.NewSessionRepo(d.DB).Get(c.Value)
	if err != nil {
		return false
	}
	return s.ExpiresAt.After(time.Now().UTC())
}

func authMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !currentSession(d, r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if configured {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			renderAdmin(w, r, "admin_setup", data)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		pw := r.FormValue("password")
		if len(pw) < minPasswordLength {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			data.Error = "Password must be at least 12 characters."
			renderAdmin(w, r, "admin_setup", data)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if pw != r.FormValue("password_confirm") {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			data.Error = "Passwords do not match."
			renderAdmin(w, r, "admin_setup", data)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hash error", http.StatusInternalServerError)
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set(passwordHashKey, string(hash)); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if err := startSession(d, w, r); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func loginHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if !configured {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			data := NewAdminPage(d, r, "login", "Sign in")
			renderAdmin(w, r, "admin_login", data)
			return
		}
		if !loginLimiter.allow(clientIP(r), loginMaxAttempts, loginWindow) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.FormValue("password"))) != nil {
			data := NewAdminPage(d, r, "login", "Sign in")
			data.Error = "Incorrect password."
			renderAdmin(w, r, "admin_login", data)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := startSession(d, w, r); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func logoutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
			_ = db.NewSessionRepo(d.DB).Delete(c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secureRequest(r),
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{entries: make(map[string]*rateEntry)}
	go func() {
		for range time.Tick(time.Minute) {
			rl.sweep()
		}
	}()
	return rl
}

func (rl *rateLimiter) sweep() {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, e := range rl.entries {
		if now.After(e.resetAt) {
			delete(rl.entries, k)
		}
	}
}

func (rl *rateLimiter) allow(key string, max int, window time.Duration) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.entries[key]
	if !ok || now.After(e.resetAt) {
		rl.entries[key] = &rateEntry{count: 1, resetAt: now.Add(window)}
		return true
	}
	if e.count >= max {
		return false
	}
	e.count++
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var loginLimiter = newRateLimiter()
```

Add the unused-import guard: `database/sql` and `strings` in the block above are only needed if used; remove any import this file does not reference.

- [ ] **Step 5: Add the login and setup templates**

Create `internal/web/templates/admin_setup.html`:

```html
{{define "admin_setup"}}<!doctype html>
<html lang="en" data-theme="{{.ThemeSlug}}" data-mode="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="robots" content="noindex">
  <title>{{.Title}} · johansen.foo</title>
  <link rel="icon" href="/static/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/static/admin.css">
</head>
<body class="auth-body">
  <main class="auth-card">
    <h1>Set up admin access</h1>
    <p class="auth-note">This password protects <code>/admin</code>. It is stored as a bcrypt hash and can be changed later.</p>
    {{if .Error}}<p class="form-error" role="alert">{{.Error}}</p>{{end}}
    <form method="post" action="/setup">
      <label for="password">Password</label>
      <input type="password" id="password" name="password" minlength="12" autocomplete="new-password" required>
      <label for="password_confirm">Confirm password</label>
      <input type="password" id="password_confirm" name="password_confirm" minlength="12" autocomplete="new-password" required>
      <button type="submit">Create password</button>
    </form>
  </main>
</body>
</html>
{{end}}
```

Create `internal/web/templates/admin_login.html`:

```html
{{define "admin_login"}}<!doctype html>
<html lang="en" data-theme="{{.ThemeSlug}}" data-mode="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="robots" content="noindex">
  <title>{{.Title}} · johansen.foo</title>
  <link rel="icon" href="/static/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/static/admin.css">
</head>
<body class="auth-body">
  <main class="auth-card">
    <h1>Sign in</h1>
    {{if .Error}}<p class="form-error" role="alert">{{.Error}}</p>{{end}}
    <form method="post" action="/login">
      <label for="password">Password</label>
      <input type="password" id="password" name="password" autocomplete="current-password" required autofocus>
      <button type="submit">Sign in</button>
    </form>
  </main>
</body>
</html>
{{end}}
```

- [ ] **Step 6: Create the minimal admin stylesheet**

Create `internal/web/static/admin.css` (Task 5 extends it):

```css
:root {
  --a-bg: #0f1117;
  --a-bg2: #161923;
  --a-bg3: #1d2130;
  --a-border: #2a2f42;
  --a-text: #e6e8ef;
  --a-muted: #9aa1b5;
  --a-accent: #7c6af7;
  --a-danger: #e5484d;
  --a-radius: 10px;
  --a-font: "JetBrains Mono", "Fira Code", ui-monospace, monospace;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  background: var(--a-bg);
  color: var(--a-text);
  font-family: var(--a-font);
  font-size: 15px;
  line-height: 1.6;
  -webkit-text-size-adjust: 100%;
}

.auth-body {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 1.5rem;
}

.auth-card {
  width: 100%;
  max-width: 26rem;
  background: var(--a-bg2);
  border: 1px solid var(--a-border);
  border-radius: var(--a-radius);
  padding: 2rem;
}

.auth-card h1 { font-size: 1.25rem; margin-bottom: 0.5rem; }
.auth-note { color: var(--a-muted); font-size: 0.8rem; margin-bottom: 1.25rem; }
.form-error {
  background: color-mix(in srgb, var(--a-danger) 18%, transparent);
  border: 1px solid var(--a-danger);
  border-radius: var(--a-radius);
  padding: 0.6rem 0.8rem;
  margin-bottom: 1rem;
  font-size: 0.85rem;
}

label { display: block; font-size: 0.8rem; color: var(--a-muted); margin-bottom: 0.3rem; }
input[type="password"], input[type="text"], input[type="url"], input[type="number"], select, textarea {
  width: 100%;
  background: var(--a-bg3);
  border: 1px solid var(--a-border);
  border-radius: var(--a-radius);
  color: var(--a-text);
  font-family: inherit;
  font-size: 0.9rem;
  padding: 0.6rem 0.7rem;
  margin-bottom: 1rem;
}
textarea { min-height: 6rem; resize: vertical; }

button {
  background: var(--a-accent);
  border: 0;
  border-radius: var(--a-radius);
  color: #fff;
  cursor: pointer;
  font-family: inherit;
  font-size: 0.9rem;
  min-height: 44px;
  padding: 0.6rem 1.1rem;
}

button.secondary { background: var(--a-bg3); border: 1px solid var(--a-border); color: var(--a-text); }
button.danger { background: var(--a-danger); }

@media (prefers-reduced-motion: reduce) {
  * { animation: none !important; transition: none !important; }
}
```

- [ ] **Step 7: Register the routes**

In `internal/web/web.go`, immediately after the existing middleware registrations and before `staticHandler`, add the CSRF cookie middleware and the two standalone auth routes. In the returned router add:

```go
	r.Get("/setup", setupHandler(d))
	r.Post("/setup", setupHandler(d))
	r.Get("/login", loginHandler(d))
	r.Post("/login", loginHandler(d))
	r.Post("/logout", logoutHandler(d))

	r.Route("/admin", func(ar chi.Router) {
		ar.Use(authMiddleware(d))
		ar.Get("/", adminDashboardHandler(d))
	})
```

`adminDashboardHandler` is created in Task 5. For this task, add a temporary minimal version in `internal/web/admin.go`:

Create `internal/web/admin.go`:

```go
package web

import "net/http"

func adminDashboardHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "dashboard", "Dashboard")
		renderAdmin(w, r, "admin_dashboard", data)
	}
}
```

and create `internal/web/templates/admin_dashboard.html` with a placeholder body that Task 5 replaces:

```html
{{define "admin_dashboard"}}<!doctype html>
<html lang="en" data-theme="{{.ThemeSlug}}" data-mode="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="robots" content="noindex">
  <title>{{.Title}} · johansen.foo</title>
  <link rel="stylesheet" href="/static/admin.css">
</head>
<body>
  <main>
    <h1>Dashboard</h1>
    <p><a href="/logout">Sign out</a></p>
  </main>
</body>
</html>
{{end}}
```

Also register the CSRF middleware in `web.go` immediately after `securityHeaders` and before `middleware.Timeout`:

```go
	r.Use(csrfMiddleware)
```

- [ ] **Step 8: Add the bcrypt dependency**

Run: `go get golang.org/x/crypto@latest && go mod tidy`
Expected: `golang.org/x/crypto` appears as a direct requirement in `go.mod`.

- [ ] **Step 9: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestSetup|TestLogin|TestAdminRequires|TestLogout' -v`
Expected: PASS for all six tests.

- [ ] **Step 10: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 11: Commit**

```bash
git add internal/web/auth.go internal/web/csrf.go internal/web/admin.go internal/web/render.go internal/web/web.go internal/web/auth_test.go internal/web/templates/admin_setup.html internal/web/templates/admin_login.html internal/web/templates/admin_dashboard.html internal/web/static/admin.css go.mod go.sum
git commit -m "feat: add bcrypt admin password, setup, login, logout and sessions"
```

---

### Task 4: CSRF protection and method override

**Files:**
- Modify: `internal/web/csrf.go`
- Modify: `internal/web/web.go` (middleware order)
- Test: `internal/web/csrf_test.go`

**Interfaces:**
- Consumes: `csrfCookieName`, `randomToken`, `secureRequest`.
- Produces: real `csrfMiddleware` and `methodOverride` middleware.

- [ ] **Step 1: Write the failing test**

Create `internal/web/csrf_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFRejectsPostWithoutToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestCSRFAcceptsMatchingToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"csrf_token": {"tok"}, "name": {"x"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("status = 403; a matching token was rejected")
	}
}

func TestCSRFSkippedForHTMXRequests(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatal("HTMX request was rejected by CSRF")
	}
}

func TestCSRFSkippedForLoginPath(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatal("/login was rejected by CSRF")
	}
}

func TestMethodOverrideRewritesPostToDelete(t *testing.T) {
	var seen string
	h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method
	}))
	form := url.Values{"_method": {"delete"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/projects/3", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != http.MethodDelete {
		t.Errorf("method = %q, want %q", seen, http.MethodDelete)
	}
}

func TestMethodOverrideLeavesPlainPostAlone(t *testing.T) {
	var seen string
	h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method
	}))
	req := httptest.NewRequest(http.MethodPost, "/admin/projects", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != http.MethodPost {
		t.Errorf("method = %q, want %q", seen, http.MethodPost)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestCSRF|TestMethodOverride' -v`
Expected: FAIL — `TestCSRFRejectsPostWithoutToken` gets 303 (the auth redirect) instead of 403 because `csrfMiddleware` is a passthrough, and `TestMethodOverride*` fail because `methodOverride` does not exist.

- [ ] **Step 3: Implement the middleware**

Replace the passthrough in `internal/web/csrf.go` with the real implementation:

```go
package web

import (
	"net/http"
	"strings"
)

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	token := randomToken()
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   secureRequest(r),
		})
	}
	return token
}

func csrfExempt(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
		return true
	}
	switch r.URL.Path {
	case "/login", "/setup":
		return true
	}
	return strings.HasPrefix(r.URL.Path, "/mcp")
}

func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if csrfExempt(r) {
			next.ServeHTTP(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || cookie.Value == "" || r.FormValue("csrf_token") != cookie.Value {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func methodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err == nil {
				if m := r.Form.Get("_method"); m != "" {
					r.Method = strings.ToUpper(m)
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Wire `methodOverride` into the router**

In `internal/web/web.go`, place it before the CSRF check so a rewritten `DELETE` is still CSRF-checked (it is a non-GET, non-HTMX-safe method):

```go
	r.Use(securityHeaders)
	r.Use(methodOverride)
	r.Use(csrfMiddleware)
	r.Use(middleware.Timeout(30 * time.Second))
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestCSRF|TestMethodOverride' -v`
Expected: PASS for all six tests.

- [ ] **Step 6: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent. `TestSetupCreatesPasswordAndSession` and `TestLogin*` still pass because `/setup` and `/login` are CSRF-exempt.

- [ ] **Step 7: Commit**

```bash
git add internal/web/csrf.go internal/web/csrf_test.go internal/web/web.go
git commit -m "feat: add CSRF protection and method override middleware"
```

---

### Task 5: Admin shell, vendored HTMX and the dashboard

**Files:**
- Create: `internal/web/static/htmx.min.js`
- Create: `internal/web/templates/admin_layout.html`
- Create: `internal/web/admin.go` (replaces the Task 3 placeholder)
- Test: `internal/web/admin_test.go`

**Interfaces:**
- Consumes: `ContentStore`, `NewAdminPage`, `renderAdmin`, `authMiddleware`.
- Produces: templates `admin_head` and `admin_foot`; `web.adminDashboardHandler(d Deps) http.HandlerFunc`.

- [ ] **Step 1: Vendor HTMX**

Run:

```bash
curl -fsSL -o internal/web/static/htmx.min.js https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js
file internal/web/static/htmx.min.js && wc -c internal/web/static/htmx.min.js
```

Expected: `file` reports JavaScript source text, and `wc -c` reports a size over 40000 bytes. If the download fails, stop and report — do not substitute a CDN URL.

- [ ] **Step 2: Write the failing test**

Create `internal/web/admin_test.go`:

```go
package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func loggedInHandler(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	sessions := db.NewSessionRepo(d)
	if _, err := sessions.Get("test-session"); err != nil {
		if err := sessions.Create("test-session", time.Now().UTC().Add(time.Hour)); err != nil {
			t.Fatalf("Create session: %v", err)
		}
	}
	return newAuthTestHandler(t, d, store)
}

func TestDashboardRendersWithVendoredHTMX(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`src="/static/htmx.min.js"`, "Dashboard", `href="/admin/projects"`, `action="/logout"`} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard body missing %q", want)
		}
	}
	if strings.Contains(body, "unpkg.com") || strings.Contains(body, "cdn.") {
		t.Error("dashboard loads a script from a third-party origin")
	}
}

func TestDashboardShowsEntityCounts(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `<span class="stat-value">9</span>`) {
		t.Errorf("dashboard does not report the project count")
	}
	if !strings.Contains(body, `<span class="stat-value">30</span>`) {
		t.Errorf("dashboard does not report the skill count")
	}
}

func TestAdminPagesAreNoIndex(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Body.String(); !strings.Contains(got, `content="noindex"`) {
		t.Error("admin page is missing robots noindex")
	}
}
```

Add `database/sql` and `time` to the import list of `admin_test.go`.

- [ ] **Step 3: Run the test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestDashboard -v`
Expected: FAIL — `src="/static/htmx.min.js"` and the count strings are absent from the placeholder dashboard.

- [ ] **Step 4: Write the layout partials**

Create `internal/web/templates/admin_layout.html`:

```html
{{define "admin_head"}}<!doctype html>
<html lang="en" data-theme="{{.ThemeSlug}}" data-mode="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
  <meta name="robots" content="noindex">
  <title>{{.Title}} · johansen.foo admin</title>
  <link rel="icon" href="/static/favicon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="/static/admin.css">
  <script src="/static/htmx.min.js" defer></script>
</head>
<body class="admin-body">
<div class="admin-shell">
  <aside class="admin-side">
    <a class="admin-brand" href="/admin">johansen.foo</a>
    <nav class="admin-nav">
      <a href="/admin"{{if eq .Section "dashboard"}} class="active"{{end}}>Dashboard</a>
      <a href="/admin/profile"{{if eq .Section "profile"}} class="active"{{end}}>Profile</a>
      <a href="/admin/social"{{if eq .Section "social"}} class="active"{{end}}>Social links</a>
      <a href="/admin/projects"{{if eq .Section "projects"}} class="active"{{end}}>Projects</a>
      <a href="/admin/experience"{{if eq .Section "experience"}} class="active"{{end}}>Experience</a>
      <a href="/admin/skills"{{if eq .Section "skills"}} class="active"{{end}}>Skills</a>
      <a href="/admin/security"{{if eq .Section "security"}} class="active"{{end}}>Security</a>
    </nav>
    <div class="admin-side-foot">
      <a href="/" target="_blank" rel="noopener">View site</a>
      <span class="admin-version">v{{.Version}}</span>
      <form method="post" action="/logout"><button class="link" type="submit">Sign out</button></form>
    </div>
  </aside>
  <main class="admin-main">
    <header class="admin-head">
      <h1>{{.Title}}</h1>
    </header>
    {{if .Flash}}<p class="form-ok" role="status">{{.Flash}}</p>{{end}}
    {{if .Error}}<p class="form-error" role="alert">{{.Error}}</p>{{end}}
{{end}}

{{define "admin_foot"}}
  </main>
</div>
</body>
</html>
{{end}}
```

- [ ] **Step 5: Extend the admin stylesheet**

Append to `internal/web/static/admin.css`:

```css
.admin-shell { display: flex; min-height: 100vh; }

.admin-side {
  width: 15rem;
  flex-shrink: 0;
  background: var(--a-bg2);
  border-right: 1px solid var(--a-border);
  padding: 1.25rem 1rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.admin-brand { color: var(--a-text); font-weight: 700; text-decoration: none; }
.admin-nav { display: flex; flex-direction: column; gap: 0.15rem; }
.admin-nav a {
  color: var(--a-muted);
  text-decoration: none;
  padding: 0.5rem 0.6rem;
  border-radius: var(--a-radius);
  font-size: 0.85rem;
}
.admin-nav a:hover { background: var(--a-bg3); color: var(--a-text); }
.admin-nav a.active { background: var(--a-bg3); color: var(--a-text); }

.admin-side-foot { margin-top: auto; display: flex; flex-direction: column; gap: 0.5rem; font-size: 0.75rem; }
.admin-side-foot a { color: var(--a-muted); }
.admin-version { color: var(--a-muted); }
button.link { background: none; border: 0; color: var(--a-muted); padding: 0; min-height: 0; text-align: left; font-size: 0.75rem; }

.admin-main { flex: 1; padding: 1.5rem 2rem 3rem; min-width: 0; }
.admin-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.5rem; }
.admin-head h1 { font-size: 1.3rem; }

.form-ok {
  background: color-mix(in srgb, var(--a-accent) 18%, transparent);
  border: 1px solid var(--a-accent);
  border-radius: var(--a-radius);
  padding: 0.6rem 0.8rem;
  margin-bottom: 1rem;
  font-size: 0.85rem;
}

.card { background: var(--a-bg2); border: 1px solid var(--a-border); border-radius: var(--a-radius); padding: 1.25rem; margin-bottom: 1.25rem; }
.card h2 { font-size: 1rem; margin-bottom: 0.75rem; }

.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr)); gap: 1rem; }
.stat { background: var(--a-bg2); border: 1px solid var(--a-border); border-radius: var(--a-radius); padding: 1rem; }
.stat-value { font-size: 1.6rem; font-weight: 700; display: block; }
.stat-label { color: var(--a-muted); font-size: 0.75rem; }

table { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
th, td { text-align: left; padding: 0.5rem 0.6rem; border-bottom: 1px solid var(--a-border); vertical-align: top; }
th { color: var(--a-muted); font-weight: 400; font-size: 0.75rem; }

.row-actions { display: flex; gap: 0.5rem; align-items: center; }
.row-actions form { display: inline; }
.actions { display: flex; gap: 0.75rem; align-items: center; margin-top: 1rem; }
.inline-form { display: flex; gap: 0.6rem; align-items: flex-end; flex-wrap: wrap; }
.inline-form > div { flex: 1; min-width: 8rem; }
.inline-form input, .inline-form select { margin-bottom: 0; }

.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 0 1rem; }

@media (max-width: 720px) {
  .admin-shell { flex-direction: column; }
  .admin-side { width: auto; flex-direction: row; flex-wrap: wrap; align-items: center; gap: 0.75rem; }
  .admin-nav { flex-direction: row; flex-wrap: wrap; }
  .admin-side-foot { margin: 0; flex-direction: row; align-items: center; }
  .admin-main { padding: 1rem; }
  .grid-2 { grid-template-columns: 1fr; }
}
```

- [ ] **Step 6: Rewrite the dashboard**

Replace the placeholder `internal/web/templates/admin_dashboard.html` with:

```html
{{define "admin_dashboard"}}{{template "admin_head" .}}
<section class="stat-grid">
  <div class="stat"><span class="stat-value">{{len .Projects}}</span><span class="stat-label">projects</span></div>
  <div class="stat"><span class="stat-value">{{len .Experience}}</span><span class="stat-label">experience entries</span></div>
  <div class="stat"><span class="stat-value">{{len .Skills}}</span><span class="stat-label">skills</span></div>
  <div class="stat"><span class="stat-value">{{len .Social}}</span><span class="stat-label">social links</span></div>
</section>

<section class="card">
  <h2>Quick actions</h2>
  <div class="actions">
    <a href="/admin/profile"><button type="button" class="secondary">Edit profile</button></a>
    <a href="/admin/projects/new"><button type="button" class="secondary">New project</button></a>
    <a href="/admin/experience/new"><button type="button" class="secondary">New experience entry</button></a>
    <a href="/admin/skills"><button type="button" class="secondary">Manage skills</button></a>
  </div>
</section>

<section class="card">
  <h2>Published content</h2>
  <p class="admin-note">Every change made here is live on the public site on the next page load. Visitor and runtime statistics arrive in a later phase.</p>
</section>
{{template "admin_foot" .}}{{end}}
```

Append `.admin-note { color: var(--a-muted); font-size: 0.8rem; }` to `admin.css`.

- [ ] **Step 7: Rewrite the dashboard handler**

Replace `internal/web/admin.go` with:

```go
package web

import "net/http"

func adminDashboardHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		data := NewAdminPage(d, r, "dashboard", "Dashboard")
		data.Profile = c.Profile
		data.Social = c.Social
		data.Projects = c.Projects
		data.Experience = c.Experience
		data.Skills = c.Skills
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_dashboard", data)
	}
}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestDashboard -v && CGO_ENABLED=0 go test ./internal/web/ -run TestAdminPagesAreNoIndex -v`
Expected: PASS.

- [ ] **Step 9: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 10: Commit**

```bash
git add internal/web/static/htmx.min.js internal/web/static/admin.css internal/web/templates/admin_layout.html internal/web/templates/admin_dashboard.html internal/web/admin.go internal/web/admin_test.go
git commit -m "feat: add the admin shell, vendored htmx and a dashboard"
```

---

### Task 6: Admin profile and social links

**Files:**
- Modify: `internal/db/repo_profile.go`
- Create: `internal/web/admin_profile.go`
- Create: `internal/web/templates/admin_profile.html`
- Create: `internal/web/templates/admin_social.html`
- Test: `internal/web/admin_profile_test.go`
- Test: `internal/db/repo_profile_test.go`

**Interfaces:**
- Consumes: `ContentStore.Reload()`, `db.ProfileRepo`.
- Produces: `(*db.ProfileRepo).Update(p *Profile) error`, `(*db.ProfileRepo).SocialLink(id int64) (*SocialLink, error)`, `(*db.ProfileRepo).CreateSocialLink(l *SocialLink) (int64, error)`, `(*db.ProfileRepo).UpdateSocialLink(l *SocialLink) error`, `(*db.ProfileRepo).DeleteSocialLink(id int64) error`; `web.adminProfileGetHandler`, `web.adminProfilePostHandler`, `web.adminSocialGetHandler`, `web.adminSocialCreateHandler`, `web.adminSocialUpdateHandler`, `web.adminSocialDeleteHandler`.

- [ ] **Step 1: Write the failing repository test**

Create `internal/db/repo_profile_test.go`:

```go
package db

import "testing"

func TestProfileRepoUpdate(t *testing.T) {
	d := seeded(t)
	r := NewProfileRepo(d)

	p, err := r.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	p.Name = "New Name"
	p.Tagline = "// new tagline"
	if err := r.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := r.Get()
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "New Name" || got.Tagline != "// new tagline" {
		t.Errorf("name=%q tagline=%q", got.Name, got.Tagline)
	}
	if got.Bio == "" {
		t.Error("Update cleared the bio column")
	}
}

func TestSocialLinkCRUD(t *testing.T) {
	d := seeded(t)
	r := NewProfileRepo(d)

	id, err := r.CreateSocialLink(&SocialLink{Platform: "example", URL: "https://example.com", Label: "Example", Sort: 99})
	if err != nil {
		t.Fatalf("CreateSocialLink: %v", err)
	}
	if id == 0 {
		t.Fatal("CreateSocialLink returned id 0")
	}

	l, err := r.SocialLink(id)
	if err != nil {
		t.Fatalf("SocialLink: %v", err)
	}
	if l.Platform != "example" || l.URL != "https://example.com" {
		t.Errorf("created link = %+v", l)
	}

	l.Label = "Renamed"
	if err := r.UpdateSocialLink(l); err != nil {
		t.Fatalf("UpdateSocialLink: %v", err)
	}
	l, _ = r.SocialLink(id)
	if l.Label != "Renamed" {
		t.Errorf("Label = %q, want %q", l.Label, "Renamed")
	}

	if err := r.DeleteSocialLink(id); err != nil {
		t.Fatalf("DeleteSocialLink: %v", err)
	}
	if _, err := r.SocialLink(id); err == nil {
		t.Error("link still readable after delete")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestProfileRepoUpdate|TestSocialLinkCRUD' -v`
Expected: FAIL — `r.Update undefined`, `r.CreateSocialLink undefined`, etc.

- [ ] **Step 3: Implement the repository methods**

Append to `internal/db/repo_profile.go`:

```go
func (r *ProfileRepo) Update(p *Profile) error {
	_, err := r.db.Exec(
		`UPDATE profile SET
		   name = ?, handle = ?, location = ?, dob = ?, tagline = ?,
		   hero_bio = ?, bio = ?, about_para_1 = ?, about_para_2 = ?, avatar = ?,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE id = 1`,
		p.Name, p.Handle, p.Location, p.DOB, p.Tagline,
		p.HeroBio, p.Bio, p.AboutPara1, p.AboutPara2, p.Avatar,
	)
	return err
}

func (r *ProfileRepo) SocialLink(id int64) (*SocialLink, error) {
	var l SocialLink
	var visible int
	err := r.db.QueryRow(
		`SELECT id, platform, url, label, sort, visible FROM social_link WHERE id = ?`, id,
	).Scan(&l.ID, &l.Platform, &l.URL, &l.Label, &l.Sort, &visible)
	if err != nil {
		return nil, err
	}
	l.Visible = visible != 0
	return &l, nil
}

func (r *ProfileRepo) CreateSocialLink(l *SocialLink) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO social_link (platform, url, label, sort, visible) VALUES (?, ?, ?, ?, ?)`,
		l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ProfileRepo) UpdateSocialLink(l *SocialLink) error {
	_, err := r.db.Exec(
		`UPDATE social_link SET platform = ?, url = ?, label = ?, sort = ?, visible = ? WHERE id = ?`,
		l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible), l.ID,
	)
	return err
}

func (r *ProfileRepo) DeleteSocialLink(id int64) error {
	_, err := r.db.Exec(`DELETE FROM social_link WHERE id = ?`, id)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
```

`SocialLink()` selects a `visible` column that `SocialLinks()` does not currently select; add `Visible bool` to `db.SocialLink` in `internal/db/models.go` and add `visible` to the `SocialLinks()` select/scan so the field is populated on list reads too. Existing callers ignore the field, so no other change is needed.

- [ ] **Step 4: Run the repository test to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestProfileRepoUpdate|TestSocialLinkCRUD' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler test**

Create `internal/web/admin_profile_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminRequest(t *testing.T, d *sql.DB, store *ContentStore, method, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	h := loggedInHandler(t, d, store)
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminProfileGetRendersCurrentValues(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/profile", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Morten Johansen", "mojoaar", "Denmark", `name="hero_bio"`, `name="bio"`} {
		if !strings.Contains(body, want) {
			t.Errorf("profile form missing %q", want)
		}
	}
}

func TestAdminProfilePostUpdatesAndReloads(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	form := url.Values{
		"name": {"Changed Name"}, "handle": {"mojoaar"}, "location": {"Denmark"},
		"dob": {"1980-08-13"}, "tagline": {"// changed"}, "hero_bio": {"hero"},
		"bio": {"bio"}, "about_para_1": {"one"}, "about_para_2": {"two"}, "avatar": {"/static/avatar.png"},
	}
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/profile", form)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	if got := store.Current().Profile.Name; got != "Changed Name" {
		t.Errorf("snapshot Profile.Name = %q, want %q", got, "Changed Name")
	}
	p, err := db.NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "Changed Name" {
		t.Errorf("stored Name = %q", p.Name)
	}
	if p.Bio != "bio" {
		t.Errorf("stored Bio = %q, want %q", p.Bio, "bio")
	}
}

func TestAdminSocialCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/social", url.Values{
		"platform": {"example"}, "url": {"https://example.com"}, "label": {"Example"}, "sort": {"99"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	links, err := db.NewProfileRepo(d).SocialLinks()
	if err != nil {
		t.Fatalf("SocialLinks: %v", err)
	}
	var id int64
	for _, l := range links {
		if l.Platform == "example" {
			id = l.ID
		}
	}
	if id == 0 {
		t.Fatal("social link was not created")
	}
	if len(store.Current().Social) != 5 {
		t.Errorf("snapshot has %d social links, want 5", len(store.Current().Social))
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/social/"+itoa(id), url.Values{
		"platform": {"example"}, "url": {"https://example.com"}, "label": {"Renamed"}, "sort": {"99"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	l, _ := db.NewProfileRepo(d).SocialLink(id)
	if l.Label != "Renamed" {
		t.Errorf("Label = %q, want %q", l.Label, "Renamed")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/social/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Social) != 4 {
		t.Errorf("snapshot has %d social links after delete, want 4", len(store.Current().Social))
	}
}
```

Add a small helper to the same file:

```go
func itoa(v int64) string { return strconv.FormatInt(v, 10) }
```

and import `strconv`, `database/sql`.

- [ ] **Step 6: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAdminProfile|TestAdminSocial' -v`
Expected: FAIL — the `/admin/profile` and `/admin/social` routes 404.

- [ ] **Step 7: Write the handlers**

Create `internal/web/admin_profile.go`:

```go
package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminProfileGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "profile", "Profile")
		data.Profile = c.Profile
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_profile", data)
	}
}

func adminProfilePostHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		p := &db.Profile{
			Name:       r.FormValue("name"),
			Handle:     r.FormValue("handle"),
			Location:   r.FormValue("location"),
			DOB:        r.FormValue("dob"),
			Tagline:    r.FormValue("tagline"),
			HeroBio:    r.FormValue("hero_bio"),
			Bio:        r.FormValue("bio"),
			AboutPara1: r.FormValue("about_para_1"),
			AboutPara2: r.FormValue("about_para_2"),
			Avatar:     r.FormValue("avatar"),
		}
		if err := db.NewProfileRepo(d.DB).Update(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/profile?saved=1", http.StatusSeeOther)
	}
}

func adminSocialGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "social", "Social links")
		data.Social = c.Social
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_social", data)
	}
}

func adminSocialCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		l := &db.SocialLink{
			Platform: r.FormValue("platform"),
			URL:      r.FormValue("url"),
			Label:    r.FormValue("label"),
			Sort:     formInt(r, "sort"),
			Visible:  r.FormValue("visible") != "",
		}
		if _, err := db.NewProfileRepo(d.DB).CreateSocialLink(l); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func adminSocialUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		l := &db.SocialLink{
			ID:       id,
			Platform: r.FormValue("platform"),
			URL:      r.FormValue("url"),
			Label:    r.FormValue("label"),
			Sort:     formInt(r, "sort"),
			Visible:  r.FormValue("visible") != "",
		}
		if err := db.NewProfileRepo(d.DB).UpdateSocialLink(l); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func adminSocialDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewProfileRepo(d.DB).DeleteSocialLink(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func formInt(r *http.Request, key string) int {
	v, err := strconv.Atoi(r.FormValue(key))
	if err != nil {
		return 0
	}
	return v
}
```

- [ ] **Step 8: Write the templates**

Create `internal/web/templates/admin_profile.html`:

```html
{{define "admin_profile"}}{{template "admin_head" .}}
<form method="post" action="/admin/profile">
  <input type="hidden" name="csrf_token" value="{{.CSRF}}">
  <section class="card">
    <h2>Identity</h2>
    <div class="grid-2">
      <div><label for="name">Name</label><input type="text" id="name" name="name" value="{{.Profile.Name}}" required></div>
      <div><label for="handle">Handle</label><input type="text" id="handle" name="handle" value="{{.Profile.Handle}}"></div>
      <div><label for="location">Location</label><input type="text" id="location" name="location" value="{{.Profile.Location}}"></div>
      <div><label for="dob">Date of birth (YYYY-MM-DD)</label><input type="text" id="dob" name="dob" value="{{.Profile.DOB}}"></div>
    </div>
    <label for="tagline">Tagline</label>
    <input type="text" id="tagline" name="tagline" value="{{.Profile.Tagline}}">
    <label for="avatar">Avatar URL</label>
    <input type="text" id="avatar" name="avatar" value="{{.Profile.Avatar}}">
  </section>

  <section class="card">
    <h2>Text</h2>
    <label for="hero_bio">Hero bio (markdown)</label>
    <textarea id="hero_bio" name="hero_bio">{{.Profile.HeroBio}}</textarea>
    <label for="bio">Machine bio for <code>/me</code> (plain text)</label>
    <textarea id="bio" name="bio">{{.Profile.Bio}}</textarea>
    <label for="about_para_1">About paragraph 1 (markdown)</label>
    <textarea id="about_para_1" name="about_para_1">{{.Profile.AboutPara1}}</textarea>
    <label for="about_para_2">About paragraph 2 (markdown)</label>
    <textarea id="about_para_2" name="about_para_2">{{.Profile.AboutPara2}}</textarea>
  </section>

  <div class="actions">
    <button type="submit">Save profile</button>
    <a href="/" target="_blank" rel="noopener"><button type="button" class="secondary">Preview site</button></a>
  </div>
</form>
{{template "admin_foot" .}}{{end}}
```

Create `internal/web/templates/admin_social.html`:

```html
{{define "admin_social"}}{{template "admin_head" .}}
<section class="card">
  <h2>Add a link</h2>
  <form method="post" action="/admin/social" class="inline-form">
    <input type="hidden" name="csrf_token" value="{{.CSRF}}">
    <div><label for="new-platform">Platform</label><input type="text" id="new-platform" name="platform" required></div>
    <div><label for="new-url">URL</label><input type="text" id="new-url" name="url" required></div>
    <div><label for="new-label">Label</label><input type="text" id="new-label" name="label"></div>
    <div><label for="new-sort">Sort</label><input type="number" id="new-sort" name="sort" value="0"></div>
    <div><label><input type="checkbox" name="visible" value="1" checked> visible</label></div>
    <button type="submit">Add</button>
  </form>
</section>

<section class="card">
  <h2>Links</h2>
  <table>
    <thead><tr><th>Platform</th><th>URL</th><th>Label</th><th>Sort</th><th>Visible</th><th></th></tr></thead>
    <tbody>
    {{range .Social}}
      <tr>
        <td colspan="6">
          <form method="post" action="/admin/social/{{.ID}}" class="inline-form">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <div><input type="text" name="platform" value="{{.Platform}}" required></div>
            <div><input type="text" name="url" value="{{.URL}}" required></div>
            <div><input type="text" name="label" value="{{.Label}}"></div>
            <div><input type="number" name="sort" value="{{.Sort}}"></div>
            <div><label><input type="checkbox" name="visible" value="1"{{if .Visible}} checked{{end}}> visible</label></div>
            <button type="submit" class="secondary">Save</button>
          </form>
        </td>
      </tr>
      <tr>
        <td colspan="6">
          <form method="post" action="/admin/social/{{.ID}}/delete" class="row-actions">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <button type="submit" class="danger">Delete {{.Platform}}</button>
          </form>
        </td>
      </tr>
    {{else}}
      <tr><td colspan="6">No social links.</td></tr>
    {{end}}
    </tbody>
  </table>
</section>
{{template "admin_foot" .}}{{end}}
```

- [ ] **Step 9: Register the routes**

In `internal/web/web.go`, inside the existing `r.Route("/admin", ...)` block, add after the dashboard route:

```go
		ar.Get("/profile", adminProfileGetHandler(d))
		ar.Post("/profile", adminProfilePostHandler(d))
		ar.Get("/social", adminSocialGetHandler(d))
		ar.Post("/social", adminSocialCreateHandler(d))
		ar.Post("/social/{id}", adminSocialUpdateHandler(d))
		ar.Post("/social/{id}/delete", adminSocialDeleteHandler(d))
```

- [ ] **Step 10: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAdminProfile|TestAdminSocial' -v`
Expected: PASS for all three tests.

- [ ] **Step 11: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 12: Commit**

```bash
git add internal/db/repo_profile.go internal/db/models.go internal/db/repo_profile_test.go internal/web/admin_profile.go internal/web/admin_profile_test.go internal/web/templates/admin_profile.html internal/web/templates/admin_social.html internal/web/web.go
git commit -m "feat: add admin CRUD for the profile and social links"
```

---

### Task 7: Admin projects

**Files:**
- Modify: `internal/db/repo_content.go`
- Create: `internal/web/admin_projects.go`
- Create: `internal/web/templates/admin_projects.html`
- Create: `internal/web/templates/admin_project_form.html`
- Test: `internal/db/repo_content_test.go`
- Test: `internal/web/admin_projects_test.go`

**Interfaces:**
- Consumes: `ContentStore.Reload()`, `db.ContentRepo`, `formInt(r, key)`, `adminRequest`.
- Produces: `(*db.ContentRepo).Project(id int64) (*Project, error)`, `CreateProject(*Project) (int64, error)`, `UpdateProject(*Project) error`, `DeleteProject(id int64) error`; handlers `adminProjectsGetHandler`, `adminProjectNewHandler`, `adminProjectEditHandler`, `adminProjectCreateHandler`, `adminProjectUpdateHandler`, `adminProjectDeleteHandler`.

- [ ] **Step 1: Write the failing repository test**

Create `internal/db/repo_content_test.go`:

```go
package db

import "testing"

func TestProjectCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateProject(&Project{
		Name: "zzz", URL: "https://example.com", Description: "desc",
		Icon: "globe", IsLink: true, URLLabel: "example.com", Sort: 99, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	p, err := r.Project(id)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Name != "zzz" || p.URL != "https://example.com" || !p.IsLink {
		t.Errorf("created = %+v", p)
	}

	p.Name = "yyy"
	p.URL = ""
	p.IsLink = false
	if err := r.UpdateProject(p); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	p, _ = r.Project(id)
	if p.Name != "yyy" || p.URL != "" || p.IsLink {
		t.Errorf("after update = %+v", p)
	}

	if err := r.DeleteProject(id); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := r.Project(id); err == nil {
		t.Error("project still readable after delete")
	}
}

func TestProjectUpdateKeepsNullURL(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	projects, err := r.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	var homelab Project
	for _, p := range projects {
		if p.Name == "homelab" {
			homelab = p
		}
	}
	if homelab.ID == 0 {
		t.Fatal("homelab project not found")
	}
	if homelab.URL != "" {
		t.Fatalf("homelab URL = %q, want empty", homelab.URL)
	}
	if err := r.UpdateProject(&homelab); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	var raw sql.NullString
	if err := d.QueryRow(`SELECT url FROM project WHERE id = ?`, homelab.ID).Scan(&raw); err != nil {
		t.Fatalf("raw select: %v", err)
	}
	if raw.Valid {
		t.Errorf("homelab url stored as %q, want NULL", raw.String)
	}
}
```

Add `database/sql` to the imports of `repo_content_test.go`.

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestProjectCRUD|TestProjectUpdateKeepsNullURL' -v`
Expected: FAIL — `r.CreateProject undefined`, `r.Project undefined`, etc.

- [ ] **Step 3: Implement the repository methods**

Append to `internal/db/repo_content.go`:

```go
func (r *ContentRepo) Project(id int64) (*Project, error) {
	var p Project
	var visible int
	var url sql.NullString
	err := r.db.QueryRow(
		`SELECT id, name, url, description, icon, is_link, url_label, sort, visible
		 FROM project WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &url, &p.Description, &p.Icon, &p.IsLink, &p.URLLabel, &p.Sort, &visible)
	if err != nil {
		return nil, err
	}
	p.URL = url.String
	p.Visible = visible != 0
	return &p, nil
}

func (r *ContentRepo) CreateProject(p *Project) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO project (name, url, description, icon, is_link, url_label, sort, visible)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, nullableURL(p.URL), p.Description, p.Icon, boolToInt(p.IsLink), p.URLLabel, p.Sort, boolToInt(p.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateProject(p *Project) error {
	_, err := r.db.Exec(
		`UPDATE project SET name = ?, url = ?, description = ?, icon = ?, is_link = ?, url_label = ?, sort = ?, visible = ?
		 WHERE id = ?`,
		p.Name, nullableURL(p.URL), p.Description, p.Icon, boolToInt(p.IsLink), p.URLLabel, p.Sort, boolToInt(p.Visible), p.ID,
	)
	return err
}

func (r *ContentRepo) DeleteProject(id int64) error {
	_, err := r.db.Exec(`DELETE FROM project WHERE id = ?`, id)
	return err
}

func nullableURL(u string) any {
	if u == "" {
		return nil
	}
	return u
}
```

`Project` needs a `Visible bool` field in `internal/db/models.go` for this to compile; add it, and add `visible` to the `Projects()` select and scan so list reads populate it. Existing callers ignore the field.

- [ ] **Step 4: Run the repository test to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestProjectCRUD|TestProjectUpdateKeepsNullURL' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler test**

Create `internal/web/admin_projects_test.go`:

```go
package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminProjectsListShowsSeededProjects(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/projects", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"atlascmdb", "mindmatrix", "/admin/projects/new"} {
		if !strings.Contains(body, want) {
			t.Errorf("project list missing %q", want)
		}
	}
}

func TestAdminProjectCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/projects", url.Values{
		"name": {"zzz"}, "url": {"https://example.com"}, "description": {"desc"},
		"icon": {"globe"}, "is_link": {"1"}, "url_label": {"example.com"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if len(store.Current().Projects) != 10 {
		t.Fatalf("snapshot projects = %d, want 10", len(store.Current().Projects))
	}

	var id int64
	for _, p := range store.Current().Projects {
		if p.Name == "zzz" {
			id = p.ID
		}
	}
	if id == 0 {
		t.Fatal("new project not found in the snapshot")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/projects/"+itoa(id), url.Values{
		"name": {"yyy"}, "url": {""}, "description": {"desc2"},
		"icon": {"globe"}, "url_label": {""}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	p, err := db.NewContentRepo(d).Project(id)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Name != "yyy" || p.IsLink {
		t.Errorf("after update = %+v", p)
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/projects/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Projects) != 9 {
		t.Errorf("snapshot projects = %d, want 9", len(store.Current().Projects))
	}
}

func TestAdminProjectEditPrefillsValues(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/projects/1", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="atlascmdb"`) {
		t.Error("edit form does not prefill the project name")
	}
}
```

- [ ] **Step 6: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminProject -v`
Expected: FAIL — the routes 404.

- [ ] **Step 7: Write the handlers**

Create `internal/web/admin_projects.go`:

```go
package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminProjectsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "projects", "Projects")
		data.Projects = c.Projects
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_projects", data)
	}
}

func adminProjectNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "projects", "New project")
		data.IsNew = true
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_project_form", data)
	}
}

func adminProjectEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		p, err := db.NewContentRepo(d.DB).Project(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		data := NewAdminPage(d, r, "projects", "Edit project")
		data.Project = *p
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_project_form", data)
	}
}

func formProject(r *http.Request) *db.Project {
	return &db.Project{
		Name:        r.FormValue("name"),
		URL:         r.FormValue("url"),
		Description: r.FormValue("description"),
		Icon:        r.FormValue("icon"),
		IsLink:      r.FormValue("url") != "",
		URLLabel:    r.FormValue("url_label"),
		Sort:        formInt(r, "sort"),
		Visible:     r.FormValue("visible") != "",
	}
}

func adminProjectCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if _, err := db.NewContentRepo(d.DB).CreateProject(formProject(r)); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}

func adminProjectUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		p := formProject(r)
		p.ID = id
		if err := db.NewContentRepo(d.DB).UpdateProject(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}

func adminProjectDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteProject(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}
```

`IsLink` is derived from a non-empty URL rather than a form checkbox: a project with no URL is not a link, and the seeded `homelab` row follows the same rule. Do not add a separate `is_link` field to the templates.

- [ ] **Step 8: Write the templates**

Create `internal/web/templates/admin_projects.html`:

```html
{{define "admin_projects"}}{{template "admin_head" .}}
<section class="card">
  <div class="actions">
    <a href="/admin/projects/new"><button type="button">New project</button></a>
  </div>
  <table>
    <thead><tr><th>Sort</th><th>Name</th><th>URL</th><th>Icon</th><th>Visible</th><th></th></tr></thead>
    <tbody>
    {{range .Projects}}
      <tr>
        <td>{{.Sort}}</td>
        <td>{{.Name}}</td>
        <td>{{if .URL}}{{.URL}}{{else}}&mdash;{{end}}</td>
        <td>{{.Icon}}</td>
        <td>{{if .Visible}}yes{{else}}no{{end}}</td>
        <td class="row-actions">
          <a href="/admin/projects/{{.ID}}"><button type="button" class="secondary">Edit</button></a>
          <form method="post" action="/admin/projects/{{.ID}}/delete">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <button type="submit" class="danger">Delete</button>
          </form>
        </td>
      </tr>
    {{else}}
      <tr><td colspan="6">No projects.</td></tr>
    {{end}}
    </tbody>
  </table>
</section>
{{template "admin_foot" .}}{{end}}
```

Create `internal/web/templates/admin_project_form.html`:

```html
{{define "admin_project_form"}}{{template "admin_head" .}}
<form method="post" action="{{if .IsNew}}/admin/projects{{else}}/admin/projects/{{.Project.ID}}{{end}}">
  <input type="hidden" name="csrf_token" value="{{.CSRF}}">
  <section class="card">
    <div class="grid-2">
      <div><label for="name">Name</label><input type="text" id="name" name="name" value="{{.Project.Name}}" required></div>
      <div><label for="icon">Icon name</label><input type="text" id="icon" name="icon" value="{{.Project.Icon}}" placeholder="globe"></div>
      <div><label for="url">URL (leave empty for a non-link card)</label><input type="text" id="url" name="url" value="{{.Project.URL}}"></div>
      <div><label for="url_label">URL label</label><input type="text" id="url_label" name="url_label" value="{{.Project.URLLabel}}"></div>
      <div><label for="sort">Sort</label><input type="number" id="sort" name="sort" value="{{.Project.Sort}}"></div>
      <div><label><input type="checkbox" name="visible" value="1"{{if or .IsNew .Project.Visible}} checked{{end}}> visible</label></div>
    </div>
    <label for="description">Description</label>
    <textarea id="description" name="description" required>{{.Project.Description}}</textarea>
  </section>
  <div class="actions">
    <button type="submit">Save</button>
    <a href="/admin/projects"><button type="button" class="secondary">Cancel</button></a>
  </div>
</form>
{{template "admin_foot" .}}{{end}}
```

Valid icon names are the 17 vendored in `internal/icons/svg/`; an unknown name renders as an empty string and the card shows no icon.

- [ ] **Step 9: Register the routes**

In `internal/web/web.go`, add to the admin route group:

```go
		ar.Get("/projects", adminProjectsGetHandler(d))
		ar.Get("/projects/new", adminProjectNewHandler(d))
		ar.Post("/projects", adminProjectCreateHandler(d))
		ar.Get("/projects/{id}", adminProjectEditHandler(d))
		ar.Post("/projects/{id}", adminProjectUpdateHandler(d))
		ar.Post("/projects/{id}/delete", adminProjectDeleteHandler(d))
```

- [ ] **Step 10: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminProject -v`
Expected: PASS for all three tests.

- [ ] **Step 11: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 12: Commit**

```bash
git add internal/db/repo_content.go internal/db/repo_content_test.go internal/web/admin_projects.go internal/web/admin_projects_test.go internal/web/templates/admin_projects.html internal/web/templates/admin_project_form.html internal/web/web.go
git commit -m "feat: add admin CRUD for projects"
```

---

### Task 8: Admin experience and skills

**Files:**
- Modify: `internal/db/repo_content.go`
- Create: `internal/web/admin_experience.go`
- Create: `internal/web/admin_skills.go`
- Create: `internal/web/templates/admin_experience.html`
- Create: `internal/web/templates/admin_experience_form.html`
- Create: `internal/web/templates/admin_skills.html`
- Test: `internal/web/admin_experience_test.go`
- Test: `internal/web/admin_skills_test.go`

**Interfaces:**
- Consumes: `ContentStore.Reload()`, `db.ContentRepo`, `formInt`, `adminRequest`, `itoa`.
- Produces: `(*db.ContentRepo).ExperienceItem(id int64) (*Experience, error)`, `CreateExperience(*Experience) (int64, error)`, `UpdateExperience(*Experience) error`, `DeleteExperience(id int64) error`, `Skill(id int64) (*Skill, error)`, `CreateSkill(*Skill) (int64, error)`, `UpdateSkill(*Skill) error`, `DeleteSkill(id int64) error`.

- [ ] **Step 1: Write the failing repository test**

Append to `internal/db/repo_content_test.go`:

```go
func TestExperienceCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateExperience(&Experience{Years: "1990-1991", Role: "Tester", Company: "ACME", Icon: "briefcase", Sort: 99, Visible: true})
	if err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}
	e, err := r.ExperienceItem(id)
	if err != nil {
		t.Fatalf("ExperienceItem: %v", err)
	}
	if e.Role != "Tester" || e.Company != "ACME" {
		t.Errorf("created = %+v", e)
	}

	e.Role = "Senior Tester"
	if err := r.UpdateExperience(e); err != nil {
		t.Fatalf("UpdateExperience: %v", err)
	}
	e, _ = r.ExperienceItem(id)
	if e.Role != "Senior Tester" {
		t.Errorf("Role = %q", e.Role)
	}

	if err := r.DeleteExperience(id); err != nil {
		t.Fatalf("DeleteExperience: %v", err)
	}
	if _, err := r.ExperienceItem(id); err == nil {
		t.Error("experience still readable after delete")
	}
}

func TestSkillCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateSkill(&Skill{Name: "Zig", Sort: 99, Visible: true})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	s, err := r.Skill(id)
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	if s.Name != "Zig" {
		t.Errorf("created = %+v", s)
	}

	s.Name = "Ziglang"
	s.Sort = 5
	if err := r.UpdateSkill(s); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	s, _ = r.Skill(id)
	if s.Name != "Ziglang" || s.Sort != 5 {
		t.Errorf("after update = %+v", s)
	}

	if err := r.DeleteSkill(id); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if _, err := r.Skill(id); err == nil {
		t.Error("skill still readable after delete")
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestExperienceCRUD|TestSkillCRUD' -v`
Expected: FAIL — the methods do not exist.

- [ ] **Step 3: Implement the repository methods**

Append to `internal/db/repo_content.go`:

```go
func (r *ContentRepo) ExperienceItem(id int64) (*Experience, error) {
	var e Experience
	var visible int
	err := r.db.QueryRow(
		`SELECT id, years, role, company, icon, sort, visible FROM experience WHERE id = ?`, id,
	).Scan(&e.ID, &e.Years, &e.Role, &e.Company, &e.Icon, &e.Sort, &visible)
	if err != nil {
		return nil, err
	}
	e.Visible = visible != 0
	return &e, nil
}

func (r *ContentRepo) CreateExperience(e *Experience) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO experience (years, role, company, icon, sort, visible) VALUES (?, ?, ?, ?, ?, ?)`,
		e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateExperience(e *Experience) error {
	_, err := r.db.Exec(
		`UPDATE experience SET years = ?, role = ?, company = ?, icon = ?, sort = ?, visible = ? WHERE id = ?`,
		e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible), e.ID,
	)
	return err
}

func (r *ContentRepo) DeleteExperience(id int64) error {
	_, err := r.db.Exec(`DELETE FROM experience WHERE id = ?`, id)
	return err
}

func (r *ContentRepo) Skill(id int64) (*Skill, error) {
	var s Skill
	var visible int
	err := r.db.QueryRow(
		`SELECT id, name, sort, visible FROM skill WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.Sort, &visible)
	if err != nil {
		return nil, err
	}
	s.Visible = visible != 0
	return &s, nil
}

func (r *ContentRepo) CreateSkill(s *Skill) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO skill (name, sort, visible) VALUES (?, ?, ?)`,
		s.Name, s.Sort, boolToInt(s.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateSkill(s *Skill) error {
	_, err := r.db.Exec(
		`UPDATE skill SET name = ?, sort = ?, visible = ? WHERE id = ?`,
		s.Name, s.Sort, boolToInt(s.Visible), s.ID,
	)
	return err
}

func (r *ContentRepo) DeleteSkill(id int64) error {
	_, err := r.db.Exec(`DELETE FROM skill WHERE id = ?`, id)
	return err
}
```

Add `Visible bool` to `db.Experience` and `db.Skill` in `internal/db/models.go`, and add `visible` to the `Experience()` and `Skills()` select/scan lists so list reads populate the field. Existing callers ignore it.

- [ ] **Step 4: Run the repository test to verify it passes**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestExperienceCRUD|TestSkillCRUD' -v`
Expected: PASS.

- [ ] **Step 5: Write the failing handler tests**

Create `internal/web/admin_experience_test.go`:

```go
package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAdminExperienceCreateAndDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/experience", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Jydske Dragonregiment") {
		t.Error("experience list is missing a seeded entry")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/experience", url.Values{
		"years": {"1990-1991"}, "role": {"Tester"}, "company": {"ACME"},
		"icon": {"briefcase"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	if len(store.Current().Experience) != 9 {
		t.Fatalf("snapshot experience = %d, want 9", len(store.Current().Experience))
	}

	var id int64
	for _, e := range store.Current().Experience {
		if e.Company == "ACME" {
			id = e.ID
		}
	}
	rec = adminRequest(t, d, store, http.MethodPost, "/admin/experience/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Experience) != 8 {
		t.Errorf("snapshot experience = %d, want 8", len(store.Current().Experience))
	}
}
```

Create `internal/web/admin_skills_test.go`:

```go
package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAdminSkillsCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/skills", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Terraform") {
		t.Error("skills list is missing a seeded skill")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills", url.Values{
		"name": {"Zig"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	if len(store.Current().Skills) != 31 {
		t.Fatalf("snapshot skills = %d, want 31", len(store.Current().Skills))
	}

	var id int64
	for _, s := range store.Current().Skills {
		if s.Name == "Zig" {
			id = s.ID
		}
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id), url.Values{
		"name": {"Ziglang"}, "sort": {"5"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	sk, err := db.NewContentRepo(d).Skill(id)
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	if sk.Name != "Ziglang" {
		t.Errorf("Name = %q, want %q", sk.Name, "Ziglang")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/skills/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Skills) != 30 {
		t.Errorf("snapshot skills = %d, want 30", len(store.Current().Skills))
	}
}
```

Add `"github.com/mojoaar/johansenfoo/internal/db"` to the imports of `admin_skills_test.go`.

- [ ] **Step 6: Run them to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAdminExperience|TestAdminSkills' -v`
Expected: FAIL — the routes 404.

- [ ] **Step 7: Write the handlers**

Create `internal/web/admin_experience.go`:

```go
package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminExperienceGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "experience", "Experience")
		data.Experience = c.Experience
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_experience", data)
	}
}

func adminExperienceNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "experience", "New experience entry")
		data.IsNew = true
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_experience_form", data)
	}
}

func adminExperienceEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		e, err := db.NewContentRepo(d.DB).ExperienceItem(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		data := NewAdminPage(d, r, "experience", "Edit experience entry")
		data.Item = *e
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_experience_form", data)
	}
}

func formExperience(r *http.Request) *db.Experience {
	return &db.Experience{
		Years:   r.FormValue("years"),
		Role:    r.FormValue("role"),
		Company: r.FormValue("company"),
		Icon:    r.FormValue("icon"),
		Sort:    formInt(r, "sort"),
		Visible: r.FormValue("visible") != "",
	}
}

func adminExperienceCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if _, err := db.NewContentRepo(d.DB).CreateExperience(formExperience(r)); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}

func adminExperienceUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		e := formExperience(r)
		e.ID = id
		if err := db.NewContentRepo(d.DB).UpdateExperience(e); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}

func adminExperienceDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteExperience(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}
```

Create `internal/web/admin_skills.go`:

```go
package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminSkillsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "skills", "Skills")
		data.Skills = c.Skills
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_skills", data)
	}
}

func adminSkillCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		s := &db.Skill{
			Name:    r.FormValue("name"),
			Sort:    formInt(r, "sort"),
			Visible: r.FormValue("visible") != "",
		}
		if _, err := db.NewContentRepo(d.DB).CreateSkill(s); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}

func adminSkillUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		s := &db.Skill{
			ID:      id,
			Name:    r.FormValue("name"),
			Sort:    formInt(r, "sort"),
			Visible: r.FormValue("visible") != "",
		}
		if err := db.NewContentRepo(d.DB).UpdateSkill(s); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}

func adminSkillDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteSkill(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}
```

- [ ] **Step 8: Write the templates**

Create `internal/web/templates/admin_experience.html`:

```html
{{define "admin_experience"}}{{template "admin_head" .}}
<section class="card">
  <div class="actions">
    <a href="/admin/experience/new"><button type="button">New experience entry</button></a>
  </div>
  <table>
    <thead><tr><th>Sort</th><th>Years</th><th>Role</th><th>Company</th><th>Icon</th><th>Visible</th><th></th></tr></thead>
    <tbody>
    {{range .Experience}}
      <tr>
        <td>{{.Sort}}</td>
        <td>{{.Years}}</td>
        <td>{{.Role}}</td>
        <td>{{.Company}}</td>
        <td>{{.Icon}}</td>
        <td>{{if .Visible}}yes{{else}}no{{end}}</td>
        <td class="row-actions">
          <a href="/admin/experience/{{.ID}}"><button type="button" class="secondary">Edit</button></a>
          <form method="post" action="/admin/experience/{{.ID}}/delete">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <button type="submit" class="danger">Delete</button>
          </form>
        </td>
      </tr>
    {{else}}
      <tr><td colspan="7">No experience entries.</td></tr>
    {{end}}
    </tbody>
  </table>
</section>
{{template "admin_foot" .}}{{end}}
```

Create `internal/web/templates/admin_experience_form.html`:

```html
{{define "admin_experience_form"}}{{template "admin_head" .}}
<form method="post" action="{{if .IsNew}}/admin/experience{{else}}/admin/experience/{{.Item.ID}}{{end}}">
  <input type="hidden" name="csrf_token" value="{{.CSRF}}">
  <section class="card">
    <div class="grid-2">
      <div><label for="years">Years</label><input type="text" id="years" name="years" value="{{.Item.Years}}" placeholder="2022–present" required></div>
      <div><label for="company">Company</label><input type="text" id="company" name="company" value="{{.Item.Company}}" required></div>
      <div><label for="role">Role</label><input type="text" id="role" name="role" value="{{.Item.Role}}" required></div>
      <div><label for="icon">Icon name</label><input type="text" id="icon" name="icon" value="{{.Item.Icon}}" placeholder="briefcase"></div>
      <div><label for="sort">Sort</label><input type="number" id="sort" name="sort" value="{{.Item.Sort}}"></div>
      <div><label><input type="checkbox" name="visible" value="1"{{if or .IsNew .Item.Visible}} checked{{end}}> visible</label></div>
    </div>
  </section>
  <div class="actions">
    <button type="submit">Save</button>
    <a href="/admin/experience"><button type="button" class="secondary">Cancel</button></a>
  </div>
</form>
{{template "admin_foot" .}}{{end}}
```

Create `internal/web/templates/admin_skills.html`:

```html
{{define "admin_skills"}}{{template "admin_head" .}}
<section class="card">
  <h2>Add a skill</h2>
  <form method="post" action="/admin/skills" class="inline-form">
    <input type="hidden" name="csrf_token" value="{{.CSRF}}">
    <div><label for="new-skill">Name</label><input type="text" id="new-skill" name="name" required></div>
    <div><label for="new-skill-sort">Sort</label><input type="number" id="new-skill-sort" name="sort" value="0"></div>
    <div><label><input type="checkbox" name="visible" value="1" checked> visible</label></div>
    <button type="submit">Add</button>
  </form>
</section>

<section class="card">
  <h2>Skills ({{len .Skills}})</h2>
  <table>
    <thead><tr><th>Name</th><th>Sort</th><th>Visible</th><th></th></tr></thead>
    <tbody>
    {{range .Skills}}
      <tr>
        <td colspan="4">
          <form method="post" action="/admin/skills/{{.ID}}" class="inline-form">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <div><input type="text" name="name" value="{{.Name}}" required></div>
            <div><input type="number" name="sort" value="{{.Sort}}"></div>
            <div><label><input type="checkbox" name="visible" value="1"{{if .Visible}} checked{{end}}> visible</label></div>
            <button type="submit" class="secondary">Save</button>
          </form>
        </td>
      </tr>
      <tr>
        <td colspan="4">
          <form method="post" action="/admin/skills/{{.ID}}/delete">
            <input type="hidden" name="csrf_token" value="{{$.CSRF}}">
            <button type="submit" class="danger">Delete {{.Name}}</button>
          </form>
        </td>
      </tr>
    {{else}}
      <tr><td colspan="4">No skills.</td></tr>
    {{end}}
    </tbody>
  </table>
</section>
{{template "admin_foot" .}}{{end}}
```

- [ ] **Step 9: Register the routes**

In `internal/web/web.go`, add to the admin route group:

```go
		ar.Get("/experience", adminExperienceGetHandler(d))
		ar.Get("/experience/new", adminExperienceNewHandler(d))
		ar.Post("/experience", adminExperienceCreateHandler(d))
		ar.Get("/experience/{id}", adminExperienceEditHandler(d))
		ar.Post("/experience/{id}", adminExperienceUpdateHandler(d))
		ar.Post("/experience/{id}/delete", adminExperienceDeleteHandler(d))
		ar.Get("/skills", adminSkillsGetHandler(d))
		ar.Post("/skills", adminSkillCreateHandler(d))
		ar.Post("/skills/{id}", adminSkillUpdateHandler(d))
		ar.Post("/skills/{id}/delete", adminSkillDeleteHandler(d))
```

- [ ] **Step 10: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAdminExperience|TestAdminSkills' -v`
Expected: PASS.

- [ ] **Step 11: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 12: Commit**

```bash
git add internal/db/repo_content.go internal/db/repo_content_test.go internal/db/models.go internal/web/admin_experience.go internal/web/admin_skills.go internal/web/admin_experience_test.go internal/web/admin_skills_test.go internal/web/templates/admin_experience.html internal/web/templates/admin_experience_form.html internal/web/templates/admin_skills.html internal/web/web.go
git commit -m "feat: add admin CRUD for experience and skills"
```

---

### Task 9: Admin security — password change and API key

**Files:**
- Create: `internal/web/admin_security.go`
- Create: `internal/web/templates/admin_security.html`
- Test: `internal/web/admin_security_test.go`

**Interfaces:**
- Consumes: `passwordHashKey`, `csrfCookieName`, `db.NewSettingsRepo`, `db.NewSessionRepo`.
- Produces: settings keys `api_key`; handlers `adminSecurityGetHandler`, `adminPasswordChangeHandler`, `adminAPIKeyRegenerateHandler`.

The API key is generated here and consumed by the Phase 3 REST API and the Phase 5 MCP server. Phase 2 only stores and displays it.

- [ ] **Step 1: Write the failing test**

Create `internal/web/admin_security_test.go`:

```go
package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func setPassword(t *testing.T, d *sql.DB, pw string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(hash)); err != nil {
		t.Fatalf("Set: %v", err)
	}
}

func TestSecurityPageHidesTheKeyWhenNoneExists(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/security", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No API key") {
		t.Error("security page does not show the empty API key state")
	}
}

func TestAPIKeyRegenerateStoresAndReveals(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/apikey", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	key, err := db.NewSettingsRepo(d).Get("api_key")
	if err != nil {
		t.Fatalf("Get api_key: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("api_key length = %d, want 64 hex chars", len(key))
	}

	rec = adminRequest(t, d, store, http.MethodGet, "/admin/security?key=1", nil)
	if !strings.Contains(rec.Body.String(), key) {
		t.Error("revealed key is not shown on the security page")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/security/apikey", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("second regenerate status = %d, want 303", rec.Code)
	}
	key2, _ := db.NewSettingsRepo(d).Get("api_key")
	if key2 == key {
		t.Error("regenerate produced the same key")
	}
}

func TestPasswordChangeRequiresCurrentPassword(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"wrong-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-long-enough-new-one"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	hash, _ := db.NewSettingsRepo(d).Get(passwordHashKey)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("original-password")) != nil {
		t.Error("password changed despite a wrong current password")
	}
}

func TestPasswordChangeSucceedsAndKeepsSession(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"original-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-long-enough-new-one"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	hash, _ := db.NewSettingsRepo(d).Get(passwordHashKey)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("a-long-enough-new-one")) != nil {
		t.Error("new password does not verify")
	}
	if _, err := db.NewSessionRepo(d).Get("test-session"); err != nil {
		t.Errorf("the acting session was destroyed: %v", err)
	}
}

func TestPasswordChangeRejectsMismatch(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"original-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-different-one-entirely"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
```

Add `database/sql` to the imports.

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestSecurity|TestAPIKey|TestPasswordChange' -v`
Expected: FAIL — the routes 404.

- [ ] **Step 3: Write the handlers**

Create `internal/web/admin_security.go`:

```go
package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const apiKeySettingKey = "api_key"

func adminSecurityGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "security", "Security")
		data.ThemeSlug = c.Theme.Slug
		if r.URL.Query().Get("key") == "1" {
			if key, err := db.NewSettingsRepo(d.DB).Get(apiKeySettingKey); err == nil {
				data.APIKey = key
			}
		}
		renderAdmin(w, r, "admin_security", data)
	}
}

func adminPasswordChangeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		current := r.FormValue("current_password")
		next := r.FormValue("new_password")

		hash, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if !configured || bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
			http.Error(w, "current password is incorrect", http.StatusBadRequest)
			return
		}
		if len(next) < minPasswordLength {
			http.Error(w, "new password is too short", http.StatusBadRequest)
			return
		}
		if next != r.FormValue("confirm_password") {
			http.Error(w, "new passwords do not match", http.StatusBadRequest)
			return
		}
		newHash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hash error", http.StatusInternalServerError)
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set(passwordHashKey, string(newHash)); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/security?saved=1", http.StatusSeeOther)
	}
}

func adminAPIKeyRegenerateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := randomToken() + randomToken()
		if err := db.NewSettingsRepo(d.DB).Set(apiKeySettingKey, key); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/security?key=1", http.StatusSeeOther)
	}
}
```

Password change deliberately keeps the acting session alive so the operator is not logged out mid-task; other sessions are not invalidated in this phase.

- [ ] **Step 4: Write the template**

Create `internal/web/templates/admin_security.html`:

```html
{{define "admin_security"}}{{template "admin_head" .}}
<section class="card">
  <h2>Change password</h2>
  <form method="post" action="/admin/security/password">
    <input type="hidden" name="csrf_token" value="{{.CSRF}}">
    <div class="grid-2">
      <div><label for="current_password">Current password</label><input type="password" id="current_password" name="current_password" autocomplete="current-password" required></div>
      <div><label for="new_password">New password</label><input type="password" id="new_password" name="new_password" minlength="12" autocomplete="new-password" required></div>
      <div><label for="confirm_password">Confirm new password</label><input type="password" id="confirm_password" name="confirm_password" minlength="12" autocomplete="new-password" required></div>
    </div>
    <div class="actions"><button type="submit">Change password</button></div>
  </form>
</section>

<section class="card">
  <h2>API key</h2>
  {{if .APIKey}}
    <p class="admin-note">Copy this now. Regenerating replaces it immediately and any client using the old key stops working.</p>
    <p><code class="api-key">{{.APIKey}}</code></p>
  {{else}}
    <p class="admin-note">No API key is configured. Agents and scripts authenticate with <code>Authorization: Bearer &lt;key&gt;</code> once one exists.</p>
  {{end}}
  <form method="post" action="/admin/security/apikey">
    <input type="hidden" name="csrf_token" value="{{.CSRF}}">
    <div class="actions">
      <button type="submit"{{if .APIKey}} class="danger"{{end}}>{{if .APIKey}}Regenerate key{{else}}Generate key{{end}}</button>
    </div>
  </form>
</section>
{{template "admin_foot" .}}{{end}}
```

Append to `admin.css`:

```css
.api-key {
  display: block;
  overflow-wrap: anywhere;
  background: var(--a-bg3);
  border: 1px solid var(--a-border);
  border-radius: var(--a-radius);
  padding: 0.6rem 0.8rem;
  font-size: 0.8rem;
  margin-bottom: 1rem;
}
```

- [ ] **Step 5: Register the routes**

In `internal/web/web.go`, add to the admin route group:

```go
		ar.Get("/security", adminSecurityGetHandler(d))
		ar.Post("/security/password", adminPasswordChangeHandler(d))
		ar.Post("/security/apikey", adminAPIKeyRegenerateHandler(d))
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestSecurity|TestAPIKey|TestPasswordChange' -v`
Expected: PASS for all five tests.

- [ ] **Step 7: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./...`
Expected: all packages `ok`, vet silent.

- [ ] **Step 8: Commit**

```bash
git add internal/web/admin_security.go internal/web/admin_security_test.go internal/web/templates/admin_security.html internal/web/static/admin.css internal/web/web.go
git commit -m "feat: add admin password change and API key management"
```

---

### Task 10: Session pruning, documentation and end-to-end verification

**Files:**
- Modify: `internal/web/web.go` (background session cleanup)
- Modify: `internal/web/render.go` (remove the dead `SiteName` field)
- Modify: `internal/web/landing.go` (construct `NewProfileRepo` once)
- Modify: `internal/web/static.go` (reject directory listings)
- Modify: `internal/web/render.go` (buffer-then-write in `renderPage` and `renderAdmin`)
- Modify: `internal/web/web_test.go` (security headers assertion)
- Modify: `cmd/johansenfoo/main.go` (nothing, unless the gate reveals a problem)
- Modify: `README.md`
- Modify: `AGENTS.md`
- Modify: `CHANGELOG.md`
- Test: `internal/web/session_cleanup_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1–9.
- Produces: no new exported surface; this task closes the Phase 2 definition of done.

- [ ] **Step 1: Write the failing cleanup test**

Create `internal/web/session_cleanup_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestPruneSessionsRemovesOnlyExpired(t *testing.T) {
	d := newTestDB(t)
	r := db.NewSessionRepo(d)
	if err := r.Create("expired", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Create expired: %v", err)
	}
	if err := r.Create("live", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create live: %v", err)
	}

	n, err := pruneSessions(d)
	if err != nil {
		t.Fatalf("pruneSessions: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned = %d, want 1", n)
	}
	if _, err := r.Get("live"); err != nil {
		t.Errorf("live session was pruned: %v", err)
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	rec := doGet(t, h, "/health")
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q", got)
	}
}

func TestStaticDirectoryListingIsRejected(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	for _, path := range []string{"/static/", "/static/fonts/"} {
		rec := doGet(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	rec := doGet(t, h, "/static/style.css")
	if rec.Code != http.StatusOK {
		t.Errorf("GET /static/style.css = %d, want 200", rec.Code)
	}
}

func doGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestPruneSessions|TestSecurityHeaders|TestStaticDirectoryListing' -v`
Expected: FAIL — `undefined: pruneSessions`; the directory-listing test gets 200 for `/static/`.

- [ ] **Step 3: Implement session pruning**

Add to `internal/web/auth.go`:

```go
func pruneSessions(d Deps) (int64, error) {
	return db.NewSessionRepo(d.DB).DeleteExpired()
}
```

Then in `internal/web/web.go`, in `New`, after the routes are registered and before `return r`, add:

```go
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			_, _ = pruneSessions(d)
		}
	}()
```

- [ ] **Step 4: Reject directory listings**

Replace `staticHandler` in `internal/web/static.go` with:

```go
func staticHandler() (http.Handler, error) {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean(r.URL.Path)
		if clean == "/" || strings.HasSuffix(clean, "/") {
			http.NotFound(w, r)
			return
		}
		if !exists(sub, strings.TrimPrefix(clean, "/")) {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return http.StripPrefix("/static/", handler), nil
}

func exists(fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return !info.IsDir()
}
```

Add `"io/fs"`, `"path"` and `"strings"` imports if they are not already present.

- [ ] **Step 5: Fix the remaining carried-forward minors**

In `internal/web/render.go`:

1. Delete the `SiteName` field from `page` and its assignment in `newPage`.
2. Change `renderPage` and `renderAdmin` to buffer before writing:

```go
func renderPage(w http.ResponseWriter, name string, data page) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}
```

and the same shape for `renderAdmin`, with `data.CSRF = ensureCSRFCookie(w, r)` moved before the buffer write.

In `internal/web/landing.go`, in `LoadContent`, keep a single `profileRepo := NewProfileRepo(d)` and reuse it for both `Get()` and `SocialLinks()`.

In `internal/web/web.go`, move `middleware.RequestID` before `middleware.Logger` so log lines carry the request id.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestPruneSessions|TestSecurityHeaders|TestStaticDirectoryListing' -v`
Expected: PASS.

- [ ] **Step 7: Update the documentation**

`README.md` — add to the feature list, and add an Admin section:

```markdown
## Admin

The site is edited through `/admin`, which does not exist until you create a password.

1. Start the service and open `http://localhost:8080/setup`.
2. Choose a password of at least 12 characters. It is stored as a bcrypt hash in the `settings` table.
3. From then on `/setup` redirects to `/login`.

Every write on an admin page reloads the content snapshot, so a change is visible on the public
site on the next page load. `/admin/security` changes the password and generates the API key that
the REST API and MCP server use (`Authorization: Bearer <key>`).
```

`AGENTS.md` — add an architecture entry for each new file and these routes:

```
Admin routes (all behind authMiddleware, HTMX-driven forms):
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

`CHANGELOG.md` — add under `## [Unreleased]`:

```markdown
### Added
- Admin surface at `/admin` with bcrypt password setup, login, logout and session cookies.
- HTMX-driven CRUD for the profile, social links, projects, experience and skills.
- Password change and regenerable API key management at `/admin/security`.
- CSRF protection on stateful requests and a login rate limiter.
- Session table with hourly pruning of expired sessions.

### Changed
- Public pages now reload site content from SQLite after every write instead of serving a
  startup snapshot, so admin edits appear on the next page load.
- Static assets reject directory listings.
```

- [ ] **Step 8: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go build ./... && gofmt -l .`
Expected: all packages `ok`, vet silent, build exit 0, `gofmt -l .` prints nothing.

- [ ] **Step 9: End-to-end manual verification**

Run the binary against a scratch data directory and exercise the real flow:

```bash
rm -rf /tmp/jf-phase2 && mkdir -p /tmp/jf-phase2
CGO_ENABLED=0 go run ./cmd/johansenfoo -data=/tmp/jf-phase2 &
sleep 2
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/                       # 200
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/admin                  # 303 -> /login
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/setup                  # 200
curl -s -c /tmp/jf-jar -b /tmp/jf-jar -X POST -d 'password=correct-horse-battery&password_confirm=correct-horse-battery' -o /dev/null -w '%{http_code}\n' http://localhost:8080/setup   # 303
curl -s -b /tmp/jf-jar -o /dev/null -w '%{http_code}\n' http://localhost:8080/admin   # 200
curl -s http://localhost:8080/ | grep -c 'Things I've built'                          # 1
kill %1
```

Then confirm the reload gate by hand: while the server is running, change the site title through `/admin/profile` with the session cookie, reload `/`, and confirm the new value appears without restarting the process. Record the observed before/after values in the report.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: prune expired sessions and close phase 2 polish items"
```

---

## Phase 2 Definition of Done

- `CGO_ENABLED=0 go test ./... -count=1` passes for every package; `CGO_ENABLED=0 go vet ./...` is silent; `gofmt -l .` prints nothing.
- `GET /setup` creates a bcrypt password and a session; afterwards `/setup` redirects to `/login`, and `/login` issues a session for the right password and returns 401 for a wrong one.
- Every `/admin/*` route redirects to `/login` without a valid session cookie.
- Stateful requests without a matching `csrf_token` and without `HX-Request: true` return 403; `/login`, `/setup` and `/mcp` are exempt.
- After any admin write, `GET /` reflects the change on the next request without a restart — verified by hand in Task 10 Step 9.
- `/admin/security` changes the password and generates a 64-character API key stored under the `api_key` setting.
- No third-party origin other than the Umami snippet; HTMX is served from `/static/htmx.min.js`.
- `README.md`, `AGENTS.md` and `CHANGELOG.md` describe the admin surface and the reload behaviour.

## Out of scope for Phase 2

Deferred to the phases named in the spec: the public and admin REST API (Phase 3), tags/drafts/RSS and the `posts_enabled` kill switch (Phase 4), the MCP server and API-key authentication middleware (Phase 5), the theme library and the theme token editor (Phase 6), visitor statistics and runtime metrics on the dashboard (Phase 7), and Docker/CI/`/docs` (Phase 8).

Two carried-forward gates remain open after this phase and must be honoured by the phase that opens the corresponding write path:

- `theme.CSS` output is still unescaped `template.CSS`. Phase 2 adds no theme write path, so it does not regress, but Phase 6 must route every DB-sourced or imported token value through `theme.Validate` and CSS/HTML-context escaping before it is interpolated into `<style>`.
- `internal/db.Open` still does not create a missing parent directory for a config-supplied `db_path`.

