package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type apiSeoResponse struct {
	SiteTitle   string       `json:"site_title"`
	TwitterSite string       `json:"twitter_site"`
	NoIndex     bool         `json:"noindex"`
	Pages       []db.PageSeo `json:"pages"`
}

func TestAdminAPISeoRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/settings/seo", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAdminAPISeoGet(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/settings/seo", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeBody[apiSeoResponse](t, rec)
	if got.SiteTitle != "Morten Johansen | johansen.foo" {
		t.Errorf("site_title = %q", got.SiteTitle)
	}
	if got.Pages == nil {
		t.Error("pages is null, want []")
	}
}

func TestAdminAPISeoPut(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	body := `{"site_title":"Put Site","twitter_site":"@put","noindex":true,"sitemap_enabled":false,"robots_txt":"User-agent: *","pages":[{"route":"/posts","title":"Writing","noindex":false}]}`
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/seo", body, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	settings := db.NewSettingsRepo(d)
	for key, want := range map[string]string{"site_title": "Put Site", "noindex": "true", "sitemap_enabled": "false"} {
		got, err := settings.Get(key)
		if err != nil || got != want {
			t.Errorf("%s = %q, err=%v; want %q", key, got, err, want)
		}
	}
	if page := store.Current().PageSeo["/posts"]; page.Title != "Writing" {
		t.Errorf("page_seo = %+v", page)
	}

	rec = apiDo(t, h, http.MethodGet, "/posts", "", nil)
	if !strings.Contains(rec.Body.String(), "<title>Writing</title>") {
		t.Error("page_seo title not applied to /posts after PUT")
	}
}

func TestAdminAPISeoRejectsEmptyRoute(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/seo",
		`{"pages":[{"route":"","title":"x"}]}`, withSession)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
