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
