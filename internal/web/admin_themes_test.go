package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func themeForm(t *testing.T, d *sql.DB, slug, name string) url.Values {
	t.Helper()
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	baseJSON, _ := json.Marshal(base.TokensBase)
	lightJSON, _ := json.Marshal(base.TokensLight)
	darkJSON, _ := json.Marshal(base.TokensDark)
	return url.Values{
		"slug":         {slug},
		"name":         {name},
		"description":  {"a test theme"},
		"tokens_base":  {string(baseJSON)},
		"tokens_light": {string(lightJSON)},
		"tokens_dark":  {string(darkJSON)},
	}
}

func TestAdminThemesList(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/themes", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Johansen") {
		t.Error("theme list missing the base theme")
	}
	if !strings.Contains(body, `name="csrf_token"`) {
		t.Error("csrf_token missing")
	}
}

func TestAdminThemeCreateAndActivate(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/themes", themeForm(t, d, "alt", "Alt"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	alt, err := db.NewThemeRepo(d).GetBySlug("alt")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/themes/"+itoa(alt.ID)+"/activate", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("activate status = %d, want 303", rec.Code)
	}

	rec = apiDo(t, newTestHandlerWith(t, d, store), http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `data-theme="alt"`) {
		t.Error("active theme not applied to the public page")
	}
}

func TestAdminThemeCreateRejectsBreakingToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	form := themeForm(t, d, "bad", "Bad")
	form.Set("tokens_dark", `{"--accent":"</style><script>alert(1)</script>"}`)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/themes", form)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if _, err := db.NewThemeRepo(d).GetBySlug("bad"); err == nil {
		t.Fatal("breaking theme was persisted")
	}
}

func TestAdminThemeDeleteGuardRail(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	base, err := db.NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/themes/"+itoa(base.ID)+"/delete", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAdminThemeDeleteNew(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	adminRequest(t, d, store, http.MethodPost, "/admin/themes", themeForm(t, d, "temp", "Temp"))
	temp, err := db.NewThemeRepo(d).GetBySlug("temp")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/themes/"+itoa(temp.ID)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if _, err := db.NewThemeRepo(d).GetBySlug("temp"); err == nil {
		t.Fatal("theme still exists after delete")
	}
}
