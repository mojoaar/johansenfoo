package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
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
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name        string  `json:"name"`
			URL         *string `json:"url"`
			Description string  `json:"description"`
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
		t.Fatalf("got %d projects, want 9", len(got.Projects))
	}
	if got.Projects[3].Name != "homelab" || got.Projects[3].URL != nil {
		t.Errorf("homelab project = %+v, want null url", got.Projects[3])
	}
	if got.Social["github"] != "https://github.com/mojoaar" {
		t.Errorf("social github = %q", got.Social["github"])
	}
}

func TestMeParityWithLegacyFileStructureOnly(t *testing.T) {
	raw, err := os.ReadFile("testdata/legacy-me.json")
	if err != nil {
		t.Fatalf("read legacy me fixture: %v", err)
	}

	var old struct {
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Bio      string            `json:"bio"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name string  `json:"name"`
			URL  *string `json:"url"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatalf("parse legacy me: %v", err)
	}

	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var got struct {
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Bio      string            `json:"bio"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name string  `json:"name"`
			URL  *string `json:"url"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if got.Name != old.Name || got.Handle != old.Handle || got.Location != old.Location || got.DOB != old.DOB {
		t.Errorf("profile fields differ: got %q/%q/%q/%q, want %q/%q/%q/%q",
			got.Name, got.Handle, got.Location, got.DOB,
			old.Name, old.Handle, old.Location, old.DOB)
	}
	if got.Bio != old.Bio {
		t.Errorf("bio differs:\n got %q\nwant %q", got.Bio, old.Bio)
	}
	if !slices.Equal(got.Skills, old.Skills) {
		t.Errorf("skills differ: got %d entries, want %d", len(got.Skills), len(old.Skills))
	}
	if len(got.Social) != len(old.Social) {
		t.Errorf("social has %d keys, want %d", len(got.Social), len(old.Social))
	}
	for k, v := range old.Social {
		if got.Social[k] != v {
			t.Errorf("social[%q] = %q, want %q", k, got.Social[k], v)
		}
	}
	if len(got.Projects) != len(old.Projects) {
		t.Fatalf("got %d projects, want %d", len(got.Projects), len(old.Projects))
	}
	for i, p := range old.Projects {
		if got.Projects[i].Name != p.Name {
			t.Errorf("project[%d].name = %q, want %q", i, got.Projects[i].Name, p.Name)
		}
		if p.Name == "homelab" {
			if p.URL != nil || got.Projects[i].URL != nil {
				t.Errorf("project[%d] (homelab) url = %v served, %v legacy, want nil on both sides",
					i, got.Projects[i].URL, p.URL)
			}
			continue
		}
		if p.URL == nil || got.Projects[i].URL == nil {
			t.Errorf("project[%d] (%s) url = %v served, %v legacy, want non-nil on both sides",
				i, p.Name, got.Projects[i].URL, p.URL)
			continue
		}
		if *got.Projects[i].URL != *p.URL {
			t.Errorf("project[%d].url = %q, want %q", i, *got.Projects[i].URL, *p.URL)
		}
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
	if !strings.Contains(body, "<loc>https://johansen.foo/posts</loc>") {
		t.Error("sitemap is missing the posts index")
	}
}

func TestSitemapIncludesPublishedPostsAndExcludesDrafts(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Live", "live-post", "published", "body", []string{"go"})
	publishPost(t, repo, "Draft", "draft-post", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil)
	body := rec.Body.String()
	if !strings.Contains(body, "<loc>https://johansen.foo/posts/live-post</loc>") {
		t.Error("published post missing from sitemap")
	}
	if !strings.Contains(body, "<loc>https://johansen.foo/tags/go</loc>") {
		t.Error("tag archive missing from sitemap")
	}
	if strings.Contains(body, "draft-post") {
		t.Error("draft leaked into sitemap")
	}
}

func TestSitemapOmitsPostsWhenDisabled(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Live", "live-post", "published", "body", nil)
	if err := db.NewSettingsRepo(d).Set("posts_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil)
	body := rec.Body.String()
	if strings.Contains(body, "live-post") {
		t.Error("posts still in sitemap while posts_enabled=false")
	}
}

func TestSitemapDisabledReturns404(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewSettingsRepo(d).Set("sitemap_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPostsKillSwitch(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Live", "live-post", "published", "body", []string{"go"})
	if err := db.NewSettingsRepo(d).Set("posts_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	for _, path := range []string{"/posts", "/posts/live-post", "/tags/go", "/feed.xml"} {
		if rec := apiDo(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, rec.Code)
		}
	}

	if err := db.NewSettingsRepo(d).Set("posts_enabled", "true"); err != nil {
		t.Fatalf("Set true: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if rec := apiDo(t, h, http.MethodGet, "/posts/live-post", "", nil); rec.Code != http.StatusOK {
		t.Errorf("after re-enable status = %d, want 200", rec.Code)
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

func TestSitemapOmitsNoindexedUrls(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	createSeoPost(t, repo, &db.Post{Slug: "hidden-post", Title: "Hidden", Status: "published", NoIndex: true})
	createSeoPost(t, repo, &db.Post{Slug: "shown-post", Title: "Shown", Status: "published"})

	rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil)
	body := rec.Body.String()
	if strings.Contains(body, "hidden-post") {
		t.Error("noindexed post advertised in sitemap")
	}
	if !strings.Contains(body, "shown-post") {
		t.Error("indexable post missing from sitemap")
	}
}

func TestSitemapUsesPageCanonical(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewPageSeoRepo(d).Upsert(&db.PageSeo{Route: "/posts", CanonicalURL: "https://example.com/writing"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil)
	body := rec.Body.String()
	if !strings.Contains(body, "<loc>https://example.com/writing</loc>") {
		t.Error("page_seo canonical not used in sitemap")
	}
	if strings.Contains(body, "<loc>https://johansen.foo/posts</loc>") {
		t.Error("default posts URL still present despite canonical override")
	}
}

func TestSitemap404WhenSiteNoindex(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewSettingsRepo(d).Set("noindex", "true"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if rec := apiDo(t, h, http.MethodGet, "/sitemap.xml", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
