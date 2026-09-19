package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func themeBody(t *testing.T, d *sql.DB, slug, name string) string {
	t.Helper()
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	body, err := json.Marshal(map[string]any{
		"slug":         slug,
		"name":         name,
		"description":  "from api",
		"tokens_base":  base.TokensBase,
		"tokens_light": base.TokensLight,
		"tokens_dark":  base.TokensDark,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(body)
}

func TestAPIAdminThemesRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/themes", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminThemesList(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/themes", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	themes := decodeBody[[]db.Theme](t, rec)
	if len(themes) == 0 || themes[0].Slug == "" {
		t.Fatalf("themes = %+v", themes)
	}
}

func TestAPIAdminThemeCreateAndActivate(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", themeBody(t, d, "api-theme", "API Theme"), withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	created := decodeBody[db.Theme](t, rec)
	if created.ID == 0 {
		t.Fatal("no id returned")
	}

	rec = apiDo(t, h, http.MethodPost, "/api/v1/admin/themes/"+itoa(created.ID)+"/activate", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	rec = apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `data-theme="api-theme"`) {
		t.Error("active theme not applied")
	}
}

func TestAPIAdminThemeCreateRejectsBadToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	body := themeBody(t, d, "bad-api", "Bad API")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	dark := parsed["tokens_dark"].(map[string]any)
	dark["--accent"] = "</style>"
	bad, _ := json.Marshal(parsed)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", string(bad), withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAPIAdminThemeDeleteGuardRail(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	rec := apiDo(t, h, http.MethodDelete, "/api/v1/admin/themes/"+itoa(base.ID), "", withSession)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestAPIAdminThemeGet(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/themes/"+itoa(base.ID), "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[db.Theme](t, rec)
	if got.Slug != "johansen" {
		t.Errorf("slug = %q", got.Slug)
	}
}

func TestAPIAdminThemeUpdateMerges(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", themeBody(t, d, "patch", "Patch"), withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d", rec.Code)
	}
	id := decodeBody[db.Theme](t, rec).ID

	rec = apiDo(t, h, http.MethodPut, "/api/v1/admin/themes/"+itoa(id),
		`{"tokens_light":{"--green":"#abcdef"}}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeBody[db.Theme](t, rec)
	if got.TokensLight["--green"] != "#abcdef" {
		t.Errorf("green not merged: %v", got.TokensLight["--green"])
	}
	if got.TokensLight["--accent"] == "" {
		t.Error("existing token lost during merge")
	}
}

func TestAPIAdminThemeDeleteNew(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", themeBody(t, d, "temp", "Temp"), withSession)
	id := decodeBody[db.Theme](t, rec).ID
	rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/themes/"+itoa(id), "", withSession)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestAPIAdminThemeRejectsSlugInjection(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	body := themeBody(t, d, `x"></style><script>alert(1)</script>`, "Evil")
	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", body, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminThemeUpdateRefusesBaseRename(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/themes/"+itoa(base.ID), `{"slug":"renamed"}`, withSession)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if _, err := db.NewThemeRepo(d).GetBySlug("johansen"); err != nil {
		t.Fatalf("johansen missing after refused rename: %v", err)
	}
}

func TestAPIAdminThemeFillsFromBase(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	body, _ := json.Marshal(map[string]any{
		"slug": "bare", "name": "Bare", "tokens_base": map[string]string{},
		"tokens_light": base.TokensLight, "tokens_dark": base.TokensDark,
	})
	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/themes", string(body), withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", rec.Code, rec.Body.String())
	}
	id := decodeBody[db.Theme](t, rec).ID
	apiDo(t, h, http.MethodPost, "/api/v1/admin/themes/"+itoa(id)+"/activate", "", withSession)

	rec = apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), "--font-sans") {
		t.Error("base tokens were not merged into the active theme")
	}
}
