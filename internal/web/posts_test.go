package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func publishPost(t *testing.T, d *db.PostRepo, title, slug, status string, body string, tags []string) int64 {
	t.Helper()
	when := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	p := &db.Post{Slug: slug, Title: title, Summary: "summary " + title, BodyMD: body, Status: status}
	if status == "published" {
		p.PublishedAt = &when
	}
	for _, name := range tags {
		p.Tags = append(p.Tags, db.Tag{Name: name, Slug: name})
	}
	id, err := d.Create(p)
	if err != nil {
		t.Fatalf("Create %s: %v", slug, err)
	}
	return id
}

func TestPostsIndexListsPublished(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Published One", "published-one", "published", "body", []string{"go"})
	publishPost(t, repo, "Secret Draft", "secret-draft", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/posts", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Published One") {
		t.Error("published post missing from index")
	}
	if strings.Contains(body, "Secret Draft") {
		t.Error("draft leaked into the post index")
	}
}

func TestPostPageRendersMarkdown(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Hello Post", "hello-post", "published",
		"# Heading\n\n```go\nfunc main() {}\n```\n", nil)

	rec := apiDo(t, h, http.MethodGet, "/posts/hello-post", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Hello Post") {
		t.Error("post title missing")
	}
	if !strings.Contains(body, "<h1") {
		t.Error("markdown heading not rendered")
	}
	if !strings.Contains(body, `class="chroma"`) {
		t.Error("code block not highlighted")
	}
}

func TestDraftPostIs404(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Draft", "a-draft", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/posts/a-draft", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestMissingPostIs404(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/posts/nope", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestNavShowsPostsWhenEnabledAndHidesWhenDisabled(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/", "", nil)
	if !strings.Contains(rec.Body.String(), `href="/posts"`) {
		t.Fatal("posts nav link missing when posts_enabled=true")
	}

	if err := db.NewSettingsRepo(d).Set("posts_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec = apiDo(t, h, http.MethodGet, "/", "", nil)
	if strings.Contains(rec.Body.String(), `href="/posts"`) {
		t.Error("posts nav link still shown when posts_enabled=false")
	}
}

func TestTagArchiveListsOnlyThatTag(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Go Post", "go-post", "published", "body", []string{"go"})
	publishPost(t, repo, "Life Post", "life-post", "published", "body", []string{"life"})

	rec := apiDo(t, h, http.MethodGet, "/tags/go", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Go Post") {
		t.Error("tag archive missing matching post")
	}
	if strings.Contains(body, "Life Post") {
		t.Error("tag archive leaked a post with a different tag")
	}
}

func TestUnknownTagIs404(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/tags/nope", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPostsIndexFiltersByTag(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Go Post", "go-post", "published", "body", []string{"go"})
	publishPost(t, repo, "Life Post", "life-post", "published", "body", []string{"life"})

	rec := apiDo(t, h, http.MethodGet, "/posts?tag=go", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Go Post") || strings.Contains(body, "Life Post") {
		t.Errorf("tag filter wrong: %s", body)
	}
}

func TestTagChipsLinkToArchive(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Go Post", "go-post", "published", "body", []string{"go"})

	rec := apiDo(t, h, http.MethodGet, "/posts", "", nil)
	if !strings.Contains(rec.Body.String(), `href="/tags/go"`) {
		t.Error("post index tag chip does not link to the tag archive")
	}
	rec = apiDo(t, h, http.MethodGet, "/posts/go-post", "", nil)
	if !strings.Contains(rec.Body.String(), `href="/tags/go"`) {
		t.Error("single post tag chip does not link to the tag archive")
	}
}
