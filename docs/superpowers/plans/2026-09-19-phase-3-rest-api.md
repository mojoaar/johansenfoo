# Phase 3 — REST API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the site a versioned REST API — unauthenticated reads for every public content type and a Bearer/session-authenticated admin surface with full CRUD plus whole-content export and import — so an agent or script can read and edit the site without the HTML admin UI.

**Architecture:** All Phase 3 changes live inside `internal/`, mostly a new set of `internal/web/api_*.go` files registered under a `/api/v1` chi subrouter. Public reads project the existing in-memory `ContentStore` snapshot (the same snapshot the HTML pages use); admin writes go through the existing `internal/db` repositories, then call `d.Content.Reload()` exactly like the HTML admin handlers. Export/import live in `internal/db/backup.go` and operate in a single SQLite transaction. No new table, no new dependency, no JavaScript.

**Tech Stack:** Go 1.25, `github.com/go-chi/chi/v5`, `modernc.org/sqlite` (CGO-free), `net/http` + `encoding/json`, `html/template` (untouched), `golang.org/x/crypto/bcrypt` (untouched).

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Public Surface, Authenticated Surface, Auth & Security, Implementation Phases §3)

## Global Constraints

Copied verbatim from the approved spec and the Phase 1/2 plans. Every task's requirements implicitly include this section.

- Module path: `github.com/mojoaar/johansenfoo`. Go directive: `go 1.25.5`.
- No CGO anywhere: every command runs as `CGO_ENABLED=0 go test ./...` / `CGO_ENABLED=0 go build ./...`.
- No comments in Go code unless essential. `//go:embed` is the only sanctioned exception.
- No third-party origin on the runtime critical path. The single exception is the existing `umami.johansen.foo` analytics snippet in `base.html`.
- All timestamps are stored and emitted as RFC 3339 UTC. In SQL that means `strftime('%Y-%m-%dT%H:%M:%SZ','now')` — never `datetime('now')`. In Go that means `time.Now().UTC().Format(time.RFC3339)`.
- `data-theme` on `<html>` is the theme SLUG; `data-mode` is `light` or `dark`. They are separate axes and must stay separate.
- All SQL is parameterized. The only non-parameterized SQL permitted is fixed string literals written by the developer (e.g. `DELETE FROM project`).
- Migrations are append-only. `0001`–`0005` must not be edited. **Phase 3 adds no migration; the next free number stays `0006` for Phase 4.**
- Every task ends with `CGO_ENABLED=0 go test ./...` and `CGO_ENABLED=0 go vet ./...` passing, then a commit.
- Documentation is part of the definition of done: a new package updates the architecture trees in `README.md` and `AGENTS.md`; changed routes update both.

## Scope of this phase (and what is deliberately deferred)

The spec's Phase 3 is "the public read endpoints and the authenticated write endpoints, plus export and import." Several endpoints the spec lists elsewhere depend on tables and packages that do not exist yet, so they are **out of scope** here and must not be stubbed:

| Endpoint | Deferred to | Reason |
| --- | --- | --- |
| `GET /api/v1/posts`, `/posts/{slug}` | Phase 4 | `post`/`tag`/`post_tag` tables do not exist |
| `GET|PUT /api/v1/admin/settings/seo` | Phase 4 | `page_seo` table does not exist |
| `/api/v1/admin/themes` CRUD, `PUT /settings/theme` value editing | Phase 6 | no theme write path; the carried-forward `</style>` escaping gate is not closed here |
| `/api/v1/admin/stats/visitors`, `/stats/system` | Phase 7 | no `page_view` table, no `internal/sysinfo` |

`PUT /api/v1/admin/settings/theme` here only selects an **already-existing** theme by slug (validation via `ThemeRepo.GetBySlug`); it writes no token values, so the Phase 1 theme-escaping gate remains deferred to Phase 6.

The spec's admin endpoint table omits social links, but the Goals require an agent to edit *every* piece of content and export/import must carry them. This plan therefore adds `GET|POST /api/v1/admin/social` and `PUT|DELETE /api/v1/admin/social/{id}` as a documented spec-gap resolution.

## Review Focus

The failure modes the spec implies but no existing test exercises, most likely first:

1. **Hidden rows leaking from public endpoints.** The repos return *all* rows plus a `Visible` flag; filtering happens in `render.go`/`me.go`/`seo.go`. A new public API handler that forgets to filter exposes drafts (Phase 4) and hidden projects/experience/social links today. Every public list handler tests exclusion of a row flipped to `visible = 0`.
2. **Unauthenticated or wrong-key writes succeeding.** `apiAuthMiddleware` is the only gate in front of admin CRUD; a missing `api_key` setting, an absent header, or an empty presented key must yield `401`, never a write or a `500`.
3. **CSRF exemption opening a cookie-based hole.** `/api/` must be exempt from the form-token CSRF middleware (API clients cannot read the HttpOnly cookie), and admin writes must reject non-JSON content types so a cross-site form POST cannot reach a handler.
4. **A failed import leaving content half-replaced.** Import must run in one transaction; a snapshot that violates a constraint must roll back completely and leave the previous content intact and the public site unchanged.
5. **Secrets leaking through export.** `admin_password_hash` and `api_key` must never appear in an export and must be ignored if present in an import payload.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/db/models.go` | Modified: JSON tags on `Profile`, `SocialLink`, `Project`, `Experience`, `Skill` so the API projects DB rows directly |
| `internal/db/backup.go` | `Snapshot`, `Export(*sql.DB)`, `Import(*sql.DB, *Snapshot)`, secret-key filtering |
| `internal/db/backup_test.go` | Export/import round-trip, atomic rollback, secret exclusion |
| `internal/web/api.go` | `writeJSON`, `writeAPIError`, `decodeJSON`, `apiID` |
| `internal/web/api_public.go` | Public read handlers for profile, projects, experience, skills, theme |
| `internal/web/api_auth.go` | `apiAuthMiddleware` — session cookie or `Authorization: Bearer <api_key>` |
| `internal/web/api_admin_profile.go` | Admin profile get/put |
| `internal/web/api_admin_social.go` | Admin social-link create/list/get/update/delete |
| `internal/web/api_admin_content.go` | Generic `crudRepo[T]` + `apiCRUDHandler[T]` + project/experience/skill adapters |
| `internal/web/api_admin_settings.go` | `PUT /settings/posts`, `PUT /settings/theme` |
| `internal/web/api_admin_backup.go` | `GET /export`, `POST /import` |
| `internal/web/api_test.go` | Shared test helpers: `apiDo`, `withSession`, `withBearer`, `decodeBody[T]` |
| `internal/web/api_public_test.go` | Public endpoint tests incl. visibility exclusion |
| `internal/web/api_auth_test.go` | 401/200 auth matrix + CSRF exemption |
| `internal/web/api_admin_profile_test.go` | Profile get/put + social CRUD |
| `internal/web/api_admin_content_test.go` | Table-driven CRUD lifecycle over all three content resources |
| `internal/web/api_admin_settings_test.go` | Settings + export/import HTTP tests |
| `internal/web/csrf.go` | Modified: exempt `/api/` paths |
| `internal/web/web.go` | Modified: register the `/api/v1` subrouter and its admin group |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Modified: new routes, new file/package, feature entry |

---

### Task 1: JSON helpers, DB model tags, and the public read API

**Files:**
- Create: `internal/web/api.go`
- Create: `internal/web/api_public.go`
- Create: `internal/web/api_test.go`
- Create: `internal/web/api_public_test.go`
- Modify: `internal/db/models.go` (add JSON tags)
- Modify: `internal/web/web.go` (register public routes)

**Interfaces:**
- Consumes: `Deps.Content *ContentStore` with `Current() *db.SiteContent`; `db.Profile`, `db.SocialLink`, `db.Project`, `db.Experience`, `db.Skill`, `db.Theme`; the existing `visibleSocial`, `visibleProjects`, `visibleExperience`, `visibleSkills` helpers from `render.go`; `theme.Resolve(base, mode map[string]string) map[string]string`.
- Produces:
  - `writeJSON(w http.ResponseWriter, status int, v any)`
  - `writeAPIError(w http.ResponseWriter, status int, msg string)`
  - `decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool`
  - `apiID(w http.ResponseWriter, r *http.Request) (int64, bool)`
  - Public handlers `apiProfileHandler`, `apiProjectsHandler`, `apiExperienceHandler`, `apiSkillsHandler`, `apiThemeHandler` (all `func(Deps) http.HandlerFunc`)
  - JSON field names on the five `db` model types (snake_case), used by export/import in Task 5.

- [ ] **Step 1: Write the failing test**

Create `internal/web/api_test.go`:

```go
package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiDo(t *testing.T, h http.Handler, method, path, body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func withSession(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
}

func withBearer(key string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body: %v; body=%q", err, rec.Body.String())
	}
	return v
}
```

Create `internal/web/api_public_test.go`:

```go
package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAPIPublicProfileIncludesSocial(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/profile", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	got := decodeBody[struct {
		Name   string            `json:"name"`
		Social []db.SocialLink   `json:"social"`
	}](t, rec)
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q, want Morten Johansen", got.Name)
	}
	if len(got.Social) != 4 {
		t.Errorf("social = %d, want 4", len(got.Social))
	}
}

func TestAPIPublicListsExcludeHiddenRows(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)

	hidden := "hidden-proj"
	if _, err := db.NewContentRepo(d).CreateProject(&db.Project{Name: hidden, Visible: false}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec := apiDo(t, h, http.MethodGet, "/api/v1/projects", "", nil)
	projects := decodeBody[[]db.Project](t, rec)
	for _, p := range projects {
		if p.Name == hidden {
			t.Fatal("hidden project leaked through the public API")
		}
	}
	if len(projects) != 9 {
		t.Errorf("projects = %d, want 9", len(projects))
	}
}

func TestAPIPublicThemeIsResolved(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/theme", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[struct {
		Slug  string            `json:"slug"`
		Light map[string]string `json:"light"`
		Dark  map[string]string `json:"dark"`
	}](t, rec)
	if got.Slug != "johansen" {
		t.Errorf("slug = %q, want johansen", got.Slug)
	}
	if got.Light["--bg"] == "" || got.Dark["--bg"] == "" {
		t.Errorf("theme tokens not resolved: light=%v dark=%v", got.Light["--bg"], got.Dark["--bg"])
	}
}
```

`newTestHandlerWith` does not exist yet — add it in `api_test.go` alongside `apiDo`:

```go
func newTestHandlerWith(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	return New(Deps{
		DB:      d,
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Content: store,
		Version: "test",
		Started: time.Now(),
	})
}
```

Add the needed imports (`database/sql`, `time`, `github.com/mojoaar/johansenfoo/internal/config`) to `api_test.go`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAPIPublic -v`
Expected: FAIL — `newTestHandlerWith` is defined but `apiProfileHandler` etc. do not exist and the routes are unregistered, so the requests return 404 (and `internal/web` may not compile if `apiDo`/`config` imports are unused; complete the helper file first).

- [ ] **Step 3: Add JSON tags to the DB models**

Modify `internal/db/models.go` so the five content types marshal as snake_case. Replace the type bodies with:

```go
type Profile struct {
	Name       string `json:"name"`
	Handle     string `json:"handle"`
	Location   string `json:"location"`
	DOB        string `json:"dob"`
	Tagline    string `json:"tagline"`
	HeroBio    string `json:"hero_bio"`
	Bio        string `json:"bio"`
	AboutPara1 string `json:"about_para_1"`
	AboutPara2 string `json:"about_para_2"`
	Avatar     string `json:"avatar"`
}

type SocialLink struct {
	ID       int64  `json:"id"`
	Platform string `json:"platform"`
	URL      string `json:"url"`
	Label    string `json:"label"`
	Sort     int    `json:"sort"`
	Visible  bool   `json:"visible"`
}

type Project struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	IsLink      bool   `json:"is_link"`
	URLLabel    string `json:"url_label"`
	Sort        int    `json:"sort"`
	Visible     bool   `json:"visible"`
}

type Experience struct {
	ID      int64  `json:"id"`
	Years   string `json:"years"`
	Role    string `json:"role"`
	Company string `json:"company"`
	Icon    string `json:"icon"`
	Sort    int    `json:"sort"`
	Visible bool   `json:"visible"`
}

type Skill struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Sort    int    `json:"sort"`
	Visible bool   `json:"visible"`
}
```

Leave `Theme` and `SiteContent` unchanged (the theme endpoint builds its own response type).

- [ ] **Step 4: Write the JSON helpers**

Create `internal/web/api.go`:

```go
package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAPIError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeAPIError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

func apiID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}
```

- [ ] **Step 5: Write the public handlers**

Create `internal/web/api_public.go`:

```go
package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

type apiProfileResponse struct {
	db.Profile
	Social []db.SocialLink `json:"social"`
}

type apiThemeResponse struct {
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Light       map[string]string `json:"light"`
	Dark        map[string]string `json:"dark"`
}

func apiProfileHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, apiProfileResponse{Profile: c.Profile, Social: visibleSocial(c.Social)})
	}
}

func apiProjectsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleProjects(c.Projects))
	}
}

func apiExperienceHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleExperience(c.Experience))
	}
}

func apiSkillsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleSkills(c.Skills))
	}
}

func apiThemeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		t := c.Theme
		writeJSON(w, http.StatusOK, apiThemeResponse{
			Slug:        t.Slug,
			Name:        t.Name,
			Description: t.Description,
			Light:       theme.Resolve(t.TokensBase, t.TokensLight),
			Dark:        theme.Resolve(t.TokensBase, t.TokensDark),
		})
	}
}
```

- [ ] **Step 6: Register the public routes**

In `internal/web/web.go`, add immediately after the `/health` registration and before the `/setup` registration:

```go
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/profile", apiProfileHandler(d))
		api.Get("/projects", apiProjectsHandler(d))
		api.Get("/experience", apiExperienceHandler(d))
		api.Get("/skills", apiSkillsHandler(d))
		api.Get("/theme", apiThemeHandler(d))
	})
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAPIPublic -v`
Expected: PASS — three tests green.

- [ ] **Step 8: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: all packages `ok`, vet silent, gofmt prints nothing. The JSON tags do not change any existing test.

- [ ] **Step 9: Commit**

```bash
git add internal/db/models.go internal/web/api.go internal/web/api_public.go internal/web/api_test.go internal/web/api_public_test.go internal/web/web.go
git commit -m "feat: add the public read API under /api/v1"
```

---

### Task 2: API authentication, CSRF exemption, and admin profile

**Files:**
- Create: `internal/web/api_auth.go`
- Create: `internal/web/api_admin_profile.go`
- Create: `internal/web/api_auth_test.go`
- Create: `internal/web/api_admin_profile_test.go`
- Modify: `internal/web/csrf.go`
- Modify: `internal/web/web.go` (register the admin group)

**Interfaces:**
- Consumes: `currentSession(d, r) bool`; `apiKeySettingKey` (const `"api_key"` from `admin_security.go`); `db.NewSettingsRepo(d.DB).Get(key)`; `writeAPIError`, `decodeJSON`; `db.NewProfileRepo(d.DB).Get()/Update(*Profile)`; `d.Content.Reload()`.
- Produces:
  - `apiAuthMiddleware(d Deps) func(http.Handler) http.Handler`
  - `apiAdminProfileGetHandler(d Deps) http.HandlerFunc`
  - `apiAdminProfilePutHandler(d Deps) http.HandlerFunc`

- [ ] **Step 1: Write the failing tests**

Create `internal/web/api_auth_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAPIAdminRequiresCredentials(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminAcceptsBearerKey(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	if err := db.NewSettingsRepo(d).Set(apiKeySettingKey, "secret-key"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	h := newTestHandlerWith(t, d, store)

	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("wrong")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status = %d, want 401", rec.Code)
	}
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("secret-key")); rec.Code != http.StatusOK {
		t.Fatalf("right key status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminRejectsBearerWhenNoKeyConfigured(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("anything"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminAcceptsSessionCookie(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestCSRFExemptsAPIPaths(t *testing.T) {
	h := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/projects", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (API paths must be CSRF-exempt)", rec.Code)
	}
}
```

Create `internal/web/api_admin_profile_test.go` (profile part only for now; Task 3 appends social tests):

```go
package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIProfileGet(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[db.Profile](t, rec)
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q", got.Name)
	}
}

func TestAdminAPIProfilePutPersistsAndReloads(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	body := `{"name":"New Name","handle":"mojoaar","location":"Denmark","dob":"1980-08-13","tagline":"t","hero_bio":"h","bio":"b","about_para_1":"a","about_para_2":"b","avatar":"avatar.png"}`
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/profile", body, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	stored, err := db.NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.Name != "New Name" {
		t.Errorf("stored name = %q, want New Name", stored.Name)
	}
	if current := store.Current(); current.Profile.Name != "New Name" {
		t.Errorf("snapshot name = %q, want New Name (Reload missing)", current.Profile.Name)
	}
}

func TestAdminAPIProfilePutRequiresName(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/profile", `{"name":""}`, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAPIAdmin|TestCSRFExemptsAPIPaths|TestAdminAPIProfile' -v`
Expected: FAIL — `/api/v1/admin/profile` is unregistered (404, not 401/200) and `apiAuthMiddleware` does not exist.

- [ ] **Step 3: Exempt `/api/` from CSRF**

In `internal/web/csrf.go`, insert into `csrfExempt`, after the HX-Request check and before the `switch`:

```go
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
```

API safety rests on two facts: browsers never attach an `Authorization` header automatically, and every admin handler requires `application/json` (Step 5), so a cross-site HTML form (which can only send `application/x-www-form-urlencoded`, `multipart/form-data`, or `text/plain`) is rejected with `415` before any write.

- [ ] **Step 4: Write the auth middleware**

Create `internal/web/api_auth.go`:

```go
package web

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

const bearerPrefix = "Bearer "

func apiAuthMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if currentSession(d, r) {
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				writeAPIError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			presented := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
			key, err := db.NewSettingsRepo(d.DB).Get(apiKeySettingKey)
			if err != nil || key == "" || presented == "" ||
				subtle.ConstantTimeCompare([]byte(presented), []byte(key)) != 1 {
				writeAPIError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 5: Write the admin profile handlers**

Create `internal/web/api_admin_profile.go`:

```go
package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminProfileGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := db.NewProfileRepo(d.DB).Get()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func apiAdminProfilePutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p db.Profile
		if !decodeJSON(w, r, &p) {
			return
		}
		if err := validateProfile(&p); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if err := repo.Update(&p); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.Get()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func validateProfile(p *db.Profile) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}
```

- [ ] **Step 6: Register the admin group**

In `internal/web/web.go`, extend the `/api/v1` block from Task 1:

```go
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/profile", apiProfileHandler(d))
		api.Get("/projects", apiProjectsHandler(d))
		api.Get("/experience", apiExperienceHandler(d))
		api.Get("/skills", apiSkillsHandler(d))
		api.Get("/theme", apiThemeHandler(d))

		api.Route("/admin", func(apiAdmin chi.Router) {
			apiAdmin.Use(apiAuthMiddleware(d))
			apiAdmin.Get("/profile", apiAdminProfileGetHandler(d))
			apiAdmin.Put("/profile", apiAdminProfilePutHandler(d))
		})
	})
```

- [ ] **Step 7: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run 'TestAPIAdmin|TestCSRFExemptsAPIPaths|TestAdminAPIProfile' -v`
Expected: PASS — all auth, CSRF, and profile tests green.

- [ ] **Step 8: Run the full gate**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: all packages `ok`, vet silent, gofmt prints nothing. `internal/web/csrf_test.go` still passes (no existing assertion expects `/api/` to require a token).

- [ ] **Step 9: Commit**

```bash
git add internal/web/api_auth.go internal/web/api_admin_profile.go internal/web/api_auth_test.go internal/web/api_admin_profile_test.go internal/web/csrf.go internal/web/web.go
git commit -m "feat: add API bearer auth and admin profile endpoint"
```

---

### Task 3: Admin social-link CRUD

**Files:**
- Create: `internal/web/api_admin_social.go`
- Modify: `internal/web/api_admin_profile_test.go` (append social tests)
- Modify: `internal/web/web.go` (register social routes)

**Interfaces:**
- Consumes: `db.NewProfileRepo(d.DB).SocialLinks()`, `.SocialLink(id)`, `.CreateSocialLink(*SocialLink)`, `.UpdateSocialLink(*SocialLink)`, `.DeleteSocialLink(id)`; `apiID`, `decodeJSON`, `writeJSON`, `writeAPIError`; `d.Content.Reload()`.
- Produces: `apiAdminSocialListHandler`, `apiAdminSocialCreateHandler`, `apiAdminSocialUpdateHandler`, `apiAdminSocialDeleteHandler` (all `func(Deps) http.HandlerFunc`).

- [ ] **Step 1: Write the failing tests**

Append to `internal/web/api_admin_profile_test.go`:

```go
func TestAdminAPISocialCRUD(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/social",
		`{"platform":"signal","url":"https://signal.me/x","label":"Signal","sort":9,"visible":true}`, withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	created := decodeBody[db.SocialLink](t, rec)
	if created.ID == 0 || created.Platform != "signal" {
		t.Fatalf("created = %+v", created)
	}

	rec = apiDo(t, h, http.MethodPut, "/api/v1/admin/social/"+itoa(created.ID),
		`{"platform":"signal","url":"https://signal.me/y","label":"Signal","sort":9,"visible":false}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	updated := decodeBody[db.SocialLink](t, rec)
	if updated.URL != "https://signal.me/y" || updated.Visible {
		t.Fatalf("updated = %+v", updated)
	}
	if current := store.Current(); !socialHidden(current.Social, created.ID) {
		t.Fatal("hidden social link still visible in the refreshed snapshot")
	}

	rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/social/"+itoa(created.ID), "", withSession)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/social/"+itoa(created.ID), "", withSession); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d, want 404", rec.Code)
	}
}

func socialHidden(links []db.SocialLink, id int64) bool {
	for _, l := range links {
		if l.ID == id {
			return !l.Visible
		}
	}
	return false
}

func TestAdminAPISocialListIncludesHidden(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/social", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decodeBody[[]db.SocialLink](t, rec); len(got) != 4 {
		t.Errorf("social = %d, want 4", len(got))
	}
}

func TestAdminAPISocialRejectsBadPayload(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/social", `not json`, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json status = %d, want 400", rec.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social", strings.NewReader("platform=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	withSession(req)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form-post status = %d, want 415", rec.Code)
	}
}
```

Add `net/http/httptest` and `strings` to that file's imports.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminAPISocial -v`
Expected: FAIL — social routes are unregistered (404 on create).

- [ ] **Step 3: Write the handlers**

Create `internal/web/api_admin_social.go`:

```go
package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminSocialListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.NewProfileRepo(d.DB).SocialLinks()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, links)
	}
}

func apiAdminSocialCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var l db.SocialLink
		if !decodeJSON(w, r, &l) {
			return
		}
		if err := validateSocial(&l); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		repo := db.NewProfileRepo(d.DB)
		id, err := repo.CreateSocialLink(&l)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		created, err := repo.SocialLink(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func apiAdminSocialUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if _, err := repo.SocialLink(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "social link not found")
			return
		}
		var l db.SocialLink
		if !decodeJSON(w, r, &l) {
			return
		}
		if err := validateSocial(&l); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		l.ID = id
		if err := repo.UpdateSocialLink(&l); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.SocialLink(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func apiAdminSocialDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if _, err := repo.SocialLink(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "social link not found")
			return
		}
		if err := repo.DeleteSocialLink(id); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateSocial(l *db.SocialLink) error {
	if strings.TrimSpace(l.Platform) == "" {
		return errors.New("platform is required")
	}
	if strings.TrimSpace(l.URL) == "" {
		return errors.New("url is required")
	}
	return nil
}
```

- [ ] **Step 4: Register the social routes**

In `internal/web/web.go`, inside the `apiAdmin` group, add:

```go
			apiAdmin.Get("/social", apiAdminSocialListHandler(d))
			apiAdmin.Post("/social", apiAdminSocialCreateHandler(d))
			apiAdmin.Put("/social/{id}", apiAdminSocialUpdateHandler(d))
			apiAdmin.Delete("/social/{id}", apiAdminSocialDeleteHandler(d))
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminAPISocial -v`
Expected: PASS — all four social tests green.

- [ ] **Step 6: Run the full gate and commit**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: all clean.

```bash
git add internal/web/api_admin_social.go internal/web/api_admin_profile_test.go internal/web/web.go
git commit -m "feat: add admin social link API"
```

---

### Task 4: Admin projects, experience, and skills CRUD

**Files:**
- Create: `internal/web/api_admin_content.go`
- Create: `internal/web/api_admin_content_test.go`
- Modify: `internal/web/web.go` (register three resource groups)

**Interfaces:**
- Consumes: `db.NewContentRepo(d.DB)` methods `Projects/Project/CreateProject/UpdateProject/DeleteProject`, `Experience/ExperienceItem/CreateExperience/UpdateExperience/DeleteExperience`, `Skills/Skill/CreateSkill/UpdateSkill/DeleteSkill`; `apiID`, `decodeJSON`, `writeJSON`, `writeAPIError`; `d.Content.Reload()`.
- Produces:
  - `type crudRepo[T any] struct { list func() ([]T, error); get func(int64) (*T, error); create func(*T) (int64, error); update func(*T) error; delete func(int64) error; setID func(*T, int64); validate func(*T) error }`
  - `apiCRUDHandler[T any](d Deps, repo crudRepo[T], resource string) http.HandlerFunc`
  - `projectCRUD(d Deps) crudRepo[db.Project]`, `experienceCRUD(d Deps) crudRepo[db.Experience]`, `skillCRUD(d Deps) crudRepo[db.Skill]`

- [ ] **Step 1: Write the failing tests**

Create `internal/web/api_admin_content_test.go`:

```go
package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIContentCRUDLifecycle(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		create     string
		update     string
		assertName func(t *testing.T, body []byte) string
	}{
		{
			name:   "projects",
			path:   "/api/v1/admin/projects",
			create: `{"name":"crud-project","description":"d","icon":"globe","visible":false,"sort":99}`,
			update: `{"name":"crud-project-2","description":"d","icon":"globe","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Project](t, body).Name
			},
		},
		{
			name:   "experience",
			path:   "/api/v1/admin/experience",
			create: `{"years":"2026","role":"crud-role","company":"c","icon":"briefcase","visible":false,"sort":99}`,
			update: `{"years":"2026","role":"crud-role-2","company":"c","icon":"briefcase","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Experience](t, body).Role
			},
		},
		{
			name:   "skills",
			path:   "/api/v1/admin/skills",
			create: `{"name":"crud-skill","visible":false,"sort":99}`,
			update: `{"name":"crud-skill-2","visible":true,"sort":99}`,
			assertName: func(t *testing.T, body []byte) string {
				return decodeJSONBytes[db.Skill](t, body).Name
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, _ := NewContentStore(d)
			h := loggedInHandler(t, d, store)

			rec := apiDo(t, h, http.MethodPost, tc.path, tc.create, withSession)
			if rec.Code != http.StatusCreated {
				t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
			}
			var created struct {
				ID int64 `json:"id"`
			}
			created = decodeBody[struct {
				ID int64 `json:"id"`
			}](t, rec)
			if created.ID == 0 {
				t.Fatal("create returned id 0")
			}
			path := tc.path + "/" + itoa(created.ID)

			rec = apiDo(t, h, http.MethodGet, path, "", withSession)
			if rec.Code != http.StatusOK {
				t.Fatalf("get status = %d, want 200", rec.Code)
			}

			rec = apiDo(t, h, http.MethodPut, path, tc.update, withSession)
			if rec.Code != http.StatusOK {
				t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if name := tc.assertName(t, rec.Body.Bytes()); name == "" {
				t.Fatal("update returned an empty name")
			}

			rec = apiDo(t, h, http.MethodDelete, path, "", withSession)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("delete status = %d, want 204", rec.Code)
			}
			if rec = apiDo(t, h, http.MethodDelete, path, "", withSession); rec.Code != http.StatusNotFound {
				t.Fatalf("second delete status = %d, want 404", rec.Code)
			}
		})
	}
}

func TestAdminAPIContentRequiresName(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	cases := []string{
		"/api/v1/admin/projects",
		"/api/v1/admin/experience",
		"/api/v1/admin/skills",
	}
	for _, path := range cases {
		rec := apiDo(t, h, http.MethodPost, path, `{}`, withSession)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", path, rec.Code)
		}
	}
}

func TestAdminAPIContentListIncludesHidden(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/projects", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := decodeBody[[]db.Project](t, rec); len(got) != 9 {
		t.Errorf("projects = %d, want 9 (all rows, including hidden)", len(got))
	}
}
```

Add a byte-slice decode helper to `internal/web/api_test.go`:

```go
func decodeJSONBytes[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode bytes: %v; body=%q", err, string(body))
	}
	return v
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminAPIContent -v`
Expected: FAIL — the three resource routes are unregistered (404 on create).

- [ ] **Step 3: Write the generic CRUD handler and adapters**

Create `internal/web/api_admin_content.go`:

```go
package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type crudRepo[T any] struct {
	list     func() ([]T, error)
	get      func(int64) (*T, error)
	create   func(*T) (int64, error)
	update   func(*T) error
	delete   func(int64) error
	setID    func(*T, int64)
	validate func(*T) error
}

func apiCRUDHandler[T any](d Deps, repo crudRepo[T], resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if chi.URLParam(r, "id") != "" {
				id, ok := apiID(w, r)
				if !ok {
					return
				}
				item, err := repo.get(id)
				if err != nil {
					writeAPIError(w, http.StatusNotFound, resource+" not found")
					return
				}
				writeJSON(w, http.StatusOK, item)
				return
			}
			items, err := repo.list()
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var item T
			if !decodeJSON(w, r, &item) {
				return
			}
			if repo.validate != nil {
				if err := repo.validate(&item); err != nil {
					writeAPIError(w, http.StatusBadRequest, err.Error())
					return
				}
			}
			id, err := repo.create(&item)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			created, err := repo.get(id)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusCreated, created)
		case http.MethodPut:
			id, ok := apiID(w, r)
			if !ok {
				return
			}
			if _, err := repo.get(id); err != nil {
				writeAPIError(w, http.StatusNotFound, resource+" not found")
				return
			}
			var item T
			if !decodeJSON(w, r, &item) {
				return
			}
			if repo.validate != nil {
				if err := repo.validate(&item); err != nil {
					writeAPIError(w, http.StatusBadRequest, err.Error())
					return
				}
			}
			repo.setID(&item, id)
			if err := repo.update(&item); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			updated, err := repo.get(id)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			id, ok := apiID(w, r)
			if !ok {
				return
			}
			if _, err := repo.get(id); err != nil {
				writeAPIError(w, http.StatusNotFound, resource+" not found")
				return
			}
			if err := repo.delete(id); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "delete failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func projectCRUD(d Deps) crudRepo[db.Project] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Project]{
		list:   r.Projects,
		get:    r.Project,
		create: r.CreateProject,
		update: r.UpdateProject,
		delete: r.DeleteProject,
		setID:  func(p *db.Project, id int64) { p.ID = id },
		validate: func(p *db.Project) error {
			if strings.TrimSpace(p.Name) == "" {
				return errors.New("name is required")
			}
			return nil
		},
	}
}

func experienceCRUD(d Deps) crudRepo[db.Experience] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Experience]{
		list:   r.Experience,
		get:    r.ExperienceItem,
		create: r.CreateExperience,
		update: r.UpdateExperience,
		delete: r.DeleteExperience,
		setID:  func(e *db.Experience, id int64) { e.ID = id },
		validate: func(e *db.Experience) error {
			if strings.TrimSpace(e.Role) == "" {
				return errors.New("role is required")
			}
			return nil
		},
	}
}

func skillCRUD(d Deps) crudRepo[db.Skill] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Skill]{
		list:   r.Skills,
		get:    r.Skill,
		create: r.CreateSkill,
		update: r.UpdateSkill,
		delete: r.DeleteSkill,
		setID:  func(s *db.Skill, id int64) { s.ID = id },
		validate: func(s *db.Skill) error {
			if strings.TrimSpace(s.Name) == "" {
				return errors.New("name is required")
			}
			return nil
		},
	}
}
```

- [ ] **Step 4: Register the three resource groups**

In `internal/web/web.go`, inside the `apiAdmin` group, add:

```go
			apiAdmin.Route("/projects", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Post("/", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Get("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Put("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Delete("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
			})
			apiAdmin.Route("/experience", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Post("/", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Get("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Put("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Delete("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
			})
			apiAdmin.Route("/skills", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Post("/", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Get("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Put("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Delete("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
			})
```

Note the trailing `/` on the collection routes. Chi serves `/api/v1/admin/projects` and `/api/v1/admin/projects/` on the `/` route; the tests use both forms (collection without slash, item with `/id`), which Chi matches correctly.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminAPIContent -v`
Expected: PASS — lifecycle for all three resources, the name-required table, and the hidden-rows list test.

- [ ] **Step 6: Run the full gate and commit**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: all clean.

```bash
git add internal/web/api_admin_content.go internal/web/api_admin_content_test.go internal/web/api_test.go internal/web/web.go
git commit -m "feat: add admin CRUD API for projects, experience and skills"
```

---

### Task 5: Settings endpoints and content export/import

**Files:**
- Create: `internal/db/backup.go`
- Create: `internal/db/backup_test.go`
- Create: `internal/web/api_admin_settings.go`
- Create: `internal/web/api_admin_backup.go`
- Create: `internal/web/api_admin_settings_test.go`
- Modify: `internal/web/web.go` (register settings and backup routes)

**Interfaces:**
- Consumes: `db.NewSettingsRepo(d.DB).Set(key, value)`; `db.NewThemeRepo(d.DB).GetBySlug(slug)`; `strconv.FormatBool`; `apiID`, `decodeJSON`, `writeJSON`, `writeAPIError`; `d.Content.Reload()`.
- Produces:
  - `db.SnapshotVersion` (const `1`), `db.Snapshot` struct, `db.Export(d *sql.DB) (*Snapshot, error)`, `db.Import(d *sql.DB, s *Snapshot) error`
  - `apiAdminPostsSettingHandler`, `apiAdminThemeSettingHandler`, `apiAdminExportHandler`, `apiAdminImportHandler`

- [ ] **Step 1: Write the failing DB tests**

Create `internal/db/backup_test.go`:

```go
package db

import (
	"testing"
)

func TestExportOmitsSecrets(t *testing.T) {
	d := seeded(t)
	if err := NewSettingsRepo(d).Set("api_key", "super-secret"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	if err := NewSettingsRepo(d).Set("admin_password_hash", "$2a$hash"); err != nil {
		t.Fatalf("Set password hash: %v", err)
	}

	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, ok := snap.Settings["api_key"]; ok {
		t.Error("api_key leaked into export")
	}
	if _, ok := snap.Settings["admin_password_hash"]; ok {
		t.Error("admin_password_hash leaked into export")
	}
	if snap.Settings["active_theme"] != "johansen" {
		t.Errorf("active_theme = %q, want johansen", snap.Settings["active_theme"])
	}
	if snap.Profile.Name == "" || len(snap.Projects) != 9 || len(snap.Skills) != 30 {
		t.Errorf("incomplete snapshot: name=%q projects=%d skills=%d",
			snap.Profile.Name, len(snap.Projects), len(snap.Skills))
	}
}

func TestImportRoundTrip(t *testing.T) {
	d := seeded(t)
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if _, err := NewContentRepo(d).CreateProject(&Project{Name: "temporary", Visible: true}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := NewProfileRepo(d).Update(&Profile{Name: "Changed"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := Import(d, snap); err != nil {
		t.Fatalf("Import: %v", err)
	}

	profile, err := NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if profile.Name != "Morten Johansen" {
		t.Errorf("name = %q, want Morten Johansen", profile.Name)
	}
	projects, err := NewContentRepo(d).Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 9 {
		t.Errorf("projects = %d, want 9", len(projects))
	}
}

func TestImportRollsBackOnFailure(t *testing.T) {
	d := seeded(t)
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	snap.Social = []SocialLink{
		{ID: 1, Platform: "a", URL: "https://a.example", Label: "A"},
		{ID: 1, Platform: "b", URL: "https://b.example", Label: "B"},
	}

	if err := Import(d, snap); err == nil {
		t.Fatal("Import succeeded with a duplicate primary key")
	}

	links, err := NewProfileRepo(d).SocialLinks()
	if err != nil {
		t.Fatalf("SocialLinks: %v", err)
	}
	if len(links) != 4 {
		t.Fatalf("social = %d, want 4 (transaction did not roll back)", len(links))
	}
	if links[0].Platform != "bluesky" {
		t.Errorf("first platform = %q, want bluesky", links[0].Platform)
	}
}

func TestImportRejectsMissingProfile(t *testing.T) {
	d := seeded(t)
	if err := Import(d, &Snapshot{Version: SnapshotVersion}); err == nil {
		t.Fatal("Import accepted a snapshot with no profile name")
	}
}
```

- [ ] **Step 2: Run the DB tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestExport|TestImport' -v`
Expected: FAIL — `Export`, `Import`, `Snapshot`, `SnapshotVersion` do not exist.

- [ ] **Step 3: Write the backup package**

Create `internal/db/backup.go`:

```go
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const SnapshotVersion = 1

var secretSettingKeys = map[string]bool{
	"admin_password_hash": true,
	"api_key":             true,
}

type Snapshot struct {
	Version    int               `json:"version"`
	ExportedAt string            `json:"exported_at"`
	Profile    Profile           `json:"profile"`
	Social     []SocialLink      `json:"social"`
	Projects   []Project         `json:"projects"`
	Experience []Experience      `json:"experience"`
	Skills     []Skill           `json:"skills"`
	Settings   map[string]string `json:"settings"`
}

func Export(d *sql.DB) (*Snapshot, error) {
	profile, err := NewProfileRepo(d).Get()
	if err != nil {
		return nil, err
	}
	social, err := NewProfileRepo(d).SocialLinks()
	if err != nil {
		return nil, err
	}
	content := NewContentRepo(d)
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
	settings, err := NewSettingsRepo(d).All()
	if err != nil {
		return nil, err
	}
	for k := range settings {
		if secretSettingKeys[k] {
			delete(settings, k)
		}
	}
	return &Snapshot{
		Version:    SnapshotVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profile:    *profile,
		Social:     social,
		Projects:   projects,
		Experience: experience,
		Skills:     skills,
		Settings:   settings,
	}, nil
}

func Import(d *sql.DB, s *Snapshot) error {
	if s == nil || s.Version != SnapshotVersion {
		return fmt.Errorf("unsupported snapshot version")
	}
	if s.Profile.Name == "" {
		return errors.New("snapshot profile name is required")
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM social_link`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM experience`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM skill`); err != nil {
		return err
	}

	p := s.Profile
	if _, err := tx.Exec(
		`UPDATE profile SET name = ?, handle = ?, location = ?, dob = ?, tagline = ?,
		   hero_bio = ?, bio = ?, about_para_1 = ?, about_para_2 = ?, avatar = ?,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE id = 1`,
		p.Name, p.Handle, p.Location, p.DOB, p.Tagline,
		p.HeroBio, p.Bio, p.AboutPara1, p.AboutPara2, p.Avatar,
	); err != nil {
		return err
	}

	for _, l := range s.Social {
		if _, err := tx.Exec(
			`INSERT INTO social_link (id, platform, url, label, sort, visible) VALUES (?, ?, ?, ?, ?, ?)`,
			l.ID, l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible),
		); err != nil {
			return err
		}
	}
	for _, pr := range s.Projects {
		if _, err := tx.Exec(
			`INSERT INTO project (id, name, url, description, icon, is_link, url_label, sort, visible)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			pr.ID, pr.Name, nullableURL(pr.URL), pr.Description, pr.Icon,
			boolToInt(pr.IsLink), pr.URLLabel, pr.Sort, boolToInt(pr.Visible),
		); err != nil {
			return err
		}
	}
	for _, e := range s.Experience {
		if _, err := tx.Exec(
			`INSERT INTO experience (id, years, role, company, icon, sort, visible) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible),
		); err != nil {
			return err
		}
	}
	for _, sk := range s.Skills {
		if _, err := tx.Exec(
			`INSERT INTO skill (id, name, sort, visible) VALUES (?, ?, ?, ?)`,
			sk.ID, sk.Name, sk.Sort, boolToInt(sk.Visible),
		); err != nil {
			return err
		}
	}
	for k, v := range s.Settings {
		if secretSettingKeys[k] {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
```

- [ ] **Step 4: Run the DB tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestExport|TestImport' -v`
Expected: PASS — four tests green.

- [ ] **Step 5: Write the failing handler tests**

Create `internal/web/api_admin_settings_test.go`:

```go
package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminAPIPostsSetting(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/posts", `{"enabled":false}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got, err := db.NewSettingsRepo(d).GetBool("posts_enabled")
	if err != nil {
		t.Fatalf("GetBool: %v", err)
	}
	if got {
		t.Error("posts_enabled = true, want false")
	}
	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/posts", `{}`, withSession); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing enabled status = %d, want 400", rec.Code)
	}
}

func TestAdminAPIThemeSetting(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/theme", `{"slug":"nope"}`, withSession); rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown theme status = %d, want 400", rec.Code)
	}
	if rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/theme", `{"slug":"johansen"}`, withSession); rec.Code != http.StatusOK {
		t.Fatalf("known theme status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got, err := db.NewSettingsRepo(d).Get("active_theme")
	if err != nil || got != "johansen" {
		t.Fatalf("active_theme = %q, err=%v", got, err)
	}
}

func TestAdminAPIExportImportRoundTrip(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/export", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("export status = %d, want 200", rec.Code)
	}
	snap := decodeBody[db.Snapshot](t, rec)
	if snap.Version != db.SnapshotVersion || snap.Profile.Name == "" {
		t.Fatalf("bad snapshot: %+v", snap)
	}

	snap.Profile.Name = "Imported Name"
	body := string(mustJSON(t, snap))
	rec = apiDo(t, h, http.MethodPost, "/api/v1/admin/import", body, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("import status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/profile", "", nil)
	got := decodeBody[db.Profile](t, rec)
	if got.Name != "Imported Name" {
		t.Errorf("public profile name = %q, want Imported Name (import did not reload)", got.Name)
	}
}
```

Add to `internal/web/api_test.go`:

```go
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
```

- [ ] **Step 6: Run the handler tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/web/ -run TestAdminAPIPostsSetting -v`
Expected: FAIL — settings routes are unregistered (404, and the test expects 200/400).

- [ ] **Step 7: Write the settings and backup handlers**

Create `internal/web/api_admin_settings.go`:

```go
package web

import (
	"net/http"
	"strconv"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type postsSettingRequest struct {
	Enabled *bool `json:"enabled"`
}

type themeSettingRequest struct {
	Slug string `json:"slug"`
}

func apiAdminPostsSettingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postsSettingRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Enabled == nil {
			writeAPIError(w, http.StatusBadRequest, "enabled is required")
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set("posts_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": *req.Enabled})
	}
}

func apiAdminThemeSettingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req themeSettingRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Slug == "" {
			writeAPIError(w, http.StatusBadRequest, "slug is required")
			return
		}
		if _, err := db.NewThemeRepo(d.DB).GetBySlug(req.Slug); err != nil {
			writeAPIError(w, http.StatusBadRequest, "unknown theme")
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set("active_theme", req.Slug); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"slug": req.Slug})
	}
}
```

Create `internal/web/api_admin_backup.go`:

```go
package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminExportHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap, err := db.Export(d.DB)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "export failed")
			return
		}
		writeJSON(w, http.StatusOK, snap)
	}
}

func apiAdminImportHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var snap db.Snapshot
		if !decodeJSON(w, r, &snap) {
			return
		}
		if err := db.Import(d.DB, &snap); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid snapshot")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
```

- [ ] **Step 8: Register the settings and backup routes**

In `internal/web/web.go`, inside the `apiAdmin` group, add:

```go
			apiAdmin.Put("/settings/posts", apiAdminPostsSettingHandler(d))
			apiAdmin.Put("/settings/theme", apiAdminThemeSettingHandler(d))
			apiAdmin.Get("/export", apiAdminExportHandler(d))
			apiAdmin.Post("/import", apiAdminImportHandler(d))
```

- [ ] **Step 9: Run the tests to verify they pass**

Run: `CGO_ENABLED=0 go test ./internal/db/ -run 'TestExport|TestImport' -v && CGO_ENABLED=0 go test ./internal/web/ -run 'TestAdminAPI' -v`
Expected: PASS — all backup and settings/backup handler tests green.

- [ ] **Step 10: Run the full gate and commit**

Run: `CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: all clean.

```bash
git add internal/db/backup.go internal/db/backup_test.go internal/web/api_admin_settings.go internal/web/api_admin_backup.go internal/web/api_admin_settings_test.go internal/web/api_test.go internal/web/web.go
git commit -m "feat: add settings API and content export/import"
```

---

### Task 6: Documentation, end-to-end verification, and definition of done

**Files:**
- Modify: `README.md`
- Modify: `AGENTS.md`
- Modify: `CHANGELOG.md`
- No new Go files.

**Interfaces:**
- Consumes: everything produced by Tasks 1–5.
- Produces: no code; a verified, documented branch ready to merge.

- [ ] **Step 1: Update the README**

Add an `## API` section after the `## Admin` section documenting:

- Base path `/api/v1`. Reads are unauthenticated; every `/api/v1/admin/*` route requires `Authorization: Bearer <api-key>` (from `/admin/security`) or a valid admin session cookie.
- Public: `GET /api/v1/profile`, `/projects`, `/experience`, `/skills`, `/theme`. Public reads only return rows with `visible = 1`.
- Admin: `GET|PUT /api/v1/admin/profile`; `GET|POST /api/v1/admin/social` and `PUT|DELETE /api/v1/admin/social/{id}`; `GET|POST /api/v1/admin/projects|experience|skills` and `GET|PUT|DELETE .../{id}`; `PUT /api/v1/admin/settings/posts`; `PUT /api/v1/admin/settings/theme`; `GET /api/v1/admin/export`; `POST /api/v1/admin/import`.
- Writes require `Content-Type: application/json`, return `201` with the created resource (create), `200` with the updated resource (update), `204` (delete), and JSON errors `{"error":"..."}`.
- Add the architecture-tree line for `internal/db/backup.go` and the `internal/web/api_*.go` group.
- Add the API to the feature list.

- [ ] **Step 2: Update AGENTS.md**

Add the API routes to the route list, and note the two invariants: public API reads filter `Visible`; every admin API write calls `d.Content.Reload()`. Add `internal/db/backup.go` to the architecture tree.

- [ ] **Step 3: Update CHANGELOG.md**

Under `## [Unreleased]` → `### Added`, add entries for the public read API, the authenticated admin API, and content export/import.

- [ ] **Step 4: Manual end-to-end verification against a scratch data dir**

```sh
CGO_ENABLED=0 go run ./cmd/johansenfoo -data=/tmp/jf-phase3 &
# create the admin password
curl -s -X POST http://localhost:8080/setup -d 'password=correcthorsebattery'
# read the public API
curl -s http://localhost:8080/api/v1/profile | head
curl -s http://localhost:8080/api/v1/theme | head
# admin API is closed without credentials
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:8080/api/v1/admin/profile   # expect 401
# generate a key in the browser at /admin/security?key=1, then:
curl -s -X PUT http://localhost:8080/api/v1/admin/settings/theme \
  -H 'Authorization: Bearer <key>' -H 'Content-Type: application/json' \
  -d '{"slug":"johansen"}'
curl -s http://localhost:8080/api/v1/admin/export \
  -H 'Authorization: Bearer <key>' > /tmp/snapshot.json
curl -s -X POST http://localhost:8080/api/v1/admin/import \
  -H 'Authorization: Bearer <key>' -H 'Content-Type: application/json' \
  --data-binary @/tmp/snapshot.json
```

Expected: public reads return JSON; the unauthenticated admin read returns `401`; the authenticated writes return `200`; export omits `admin_password_hash` and `api_key`; re-import leaves the site unchanged.

- [ ] **Step 5: Run the full gate one final time**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./... -count=1 && CGO_ENABLED=0 go vet ./... && gofmt -l .`
Expected: build OK, all packages `ok`, vet silent, gofmt prints nothing.

- [ ] **Step 6: Commit**

```bash
git add README.md AGENTS.md CHANGELOG.md
git commit -m "docs: document the phase 3 REST API"
```

---

## Phase 3 Definition of Done

- `GET /api/v1/profile|projects|experience|skills|theme` return JSON and exclude hidden rows.
- Every `/api/v1/admin/*` route returns `401` without a valid session cookie or Bearer API key, and works with either.
- Admin CRUD for profile, social links, projects, experience, and skills persists and is visible on the public site on the next request (Reload gate proven by test).
- `PUT /settings/posts` and `PUT /settings/theme` persist and validate; unknown theme slugs are rejected.
- `GET /export` produces a snapshot with no `admin_password_hash` or `api_key`; `POST /import` restores content transactionally and reloads the snapshot.
- `CGO_ENABLED=0 go build ./...`, `go test ./... -count=1`, `go vet ./...`, and `gofmt -l .` all pass.
- README, AGENTS.md, and CHANGELOG document the new surface.

## Out of scope for Phase 3

- Posts, tags, RSS, markdown rendering pipeline, Chroma, `posts_enabled` read behaviour (Phase 4).
- `page_seo` overrides and SEO settings endpoints (Phase 4).
- MCP server and tools (Phase 5).
- Theme token CRUD, theme library, `import_theme` (Phase 6).
- Visitor and runtime statistics (Phase 7).
- Dockerfile, compose, CI, `/docs` reference page (Phase 8).
- Any page-view recording, rate limiting on the API, and caching headers for API responses (deferred; the spec sets MCP rate limiting at 100/min, not the REST API).
