package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func createSeoPost(t *testing.T, d *db.PostRepo, p *db.Post) {
	t.Helper()
	if p.Status == "" {
		p.Status = "published"
	}
	if p.Status == "published" && p.PublishedAt == nil {
		when := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		p.PublishedAt = &when
	}
	if _, err := d.Create(p); err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestPageSeoOverridesPostIndexTitle(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewPageSeoRepo(d).Upsert(&db.PageSeo{Route: "/posts", Title: "Custom Writing"}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec := apiDo(t, h, http.MethodGet, "/posts", "", nil)
	if !strings.Contains(rec.Body.String(), "<title>Custom Writing</title>") {
		t.Errorf("page_seo title not applied: %s", rec.Body.String()[:200])
	}
}

func TestPostSEOTitleUsesTemplate(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	createSeoPost(t, db.NewPostRepo(d), &db.Post{
		Slug: "hello", Title: "Hello", SEOTitle: "My Custom Title", Status: "published",
	})

	rec := apiDo(t, h, http.MethodGet, "/posts/hello", "", nil)
	if !strings.Contains(rec.Body.String(), "<title>My Custom Title | johansen.foo</title>") {
		t.Errorf("post seo_title not templated")
	}
}

func TestPostSEODescriptionFallsBackToSummary(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	createSeoPost(t, repo, &db.Post{Slug: "a", Title: "A", Summary: "the summary", Status: "published"})

	rec := apiDo(t, h, http.MethodGet, "/posts/a", "", nil)
	if !strings.Contains(rec.Body.String(), `content="the summary"`) {
		t.Error("summary not used as the post description")
	}

	createSeoPost(t, repo, &db.Post{Slug: "b", Title: "B", Summary: "the summary", SEODescription: "custom desc", Status: "published"})
	rec = apiDo(t, h, http.MethodGet, "/posts/b", "", nil)
	if !strings.Contains(rec.Body.String(), `content="custom desc"`) {
		t.Error("post seo_description did not beat the summary")
	}
}

func TestSiteNoindexMeta(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewSettingsRepo(d).Set("noindex", "true"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `name="robots" content="noindex, nofollow"`) {
		t.Error("site-wide noindex not applied")
	}
}

func TestPageSeoNoindexMeta(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewPageSeoRepo(d).Upsert(&db.PageSeo{Route: "/posts", NoIndex: true}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := apiDo(t, h, http.MethodGet, "/posts", "", nil)
	if !strings.Contains(rec.Body.String(), `name="robots" content="noindex, nofollow"`) {
		t.Error("page_seo noindex not applied")
	}
}

func TestTwitterSiteRenderedWhenSet(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/", "", nil)
	if strings.Contains(rec.Body.String(), "twitter:site") {
		t.Error("twitter:site rendered while unset")
	}

	if err := db.NewSettingsRepo(d).Set("twitter_site", "@mojoaar"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec = apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `name="twitter:site" content="@mojoaar"`) {
		t.Error("twitter:site not rendered when set")
	}
}

func TestPostPageEmitsBlogPosting(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	createSeoPost(t, db.NewPostRepo(d), &db.Post{
		Slug: "bp", Title: "Blog Posting Test", Summary: "sum", Status: "published",
	})

	rec := apiDo(t, h, http.MethodGet, "/posts/bp", "", nil)
	body := rec.Body.String()
	if !strings.Contains(body, `"@type": "BlogPosting"`) {
		t.Error("BlogPosting schema missing")
	}
	if !strings.Contains(body, `"headline": "Blog Posting Test"`) {
		t.Error("BlogPosting headline missing")
	}
	if !strings.Contains(body, `"datePublished"`) {
		t.Error("BlogPosting datePublished missing")
	}

	rec = apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `"@type": "Person"`) {
		t.Error("Person schema lost from the landing page")
	}
}
