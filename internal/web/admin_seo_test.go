package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminSeoGetRendersGlobals(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/seo", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Morten Johansen | johansen.foo") {
		t.Error("site_title not rendered")
	}
	if !strings.Contains(body, `name="csrf_token"`) {
		t.Error("csrf_token field missing")
	}
}

func TestAdminSeoPostSavesGlobals(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/seo", url.Values{
		"site_title":     {"My Site"},
		"title_template": {"%s :: My Site"},
		"twitter_site":   {"@mojoaar"},
		"robots_txt":     {"User-agent: *\nDisallow:\n"},
		"noindex":        {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	settings := db.NewSettingsRepo(d)
	for key, want := range map[string]string{
		"site_title": "My Site", "twitter_site": "@mojoaar", "noindex": "true",
	} {
		got, err := settings.Get(key)
		if err != nil || got != want {
			t.Errorf("%s = %q, err=%v; want %q", key, got, err, want)
		}
	}
	if got := store.Current().Settings["site_title"]; got != "My Site" {
		t.Errorf("snapshot site_title = %q (Reload missing)", got)
	}
}

func TestAdminSeoPageUpsert(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/seo/pages", url.Values{
		"route":       {"/posts"},
		"title":       {"Writing"},
		"description": {"posts desc"},
		"noindex":     {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	got := store.Current().PageSeo["/posts"]
	if got.Title != "Writing" || !got.NoIndex {
		t.Fatalf("page_seo = %+v", got)
	}
}

func TestAdminSeoPageDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	if err := db.NewPageSeoRepo(d).Upsert(&db.PageSeo{Route: "/", Title: "Home"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	page, _ := db.NewPageSeoRepo(d).Get("/")

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/seo/pages/"+itoa(page.ID)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if _, ok := store.Current().PageSeo["/"]; ok {
		t.Error("page_seo survived delete")
	}
}
