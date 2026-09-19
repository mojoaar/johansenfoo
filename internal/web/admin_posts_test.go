package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminPostsList(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	publishPost(t, db.NewPostRepo(d), "Listed Post", "listed-post", "published", "body", nil)

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/posts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Listed Post") {
		t.Error("admin post list is missing the post")
	}
}

func TestAdminPostCreate(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts", url.Values{
		"title":   {"New Post"},
		"slug":    {"new-post"},
		"summary": {"a summary"},
		"body_md": {"# Body"},
		"status":  {"published"},
		"tags":    {"go, testing"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	p, err := db.NewPostRepo(d).PublishedBySlug("new-post")
	if err != nil {
		t.Fatalf("PublishedBySlug: %v", err)
	}
	if p.Title != "New Post" || p.Summary != "a summary" || p.BodyMD != "# Body" {
		t.Errorf("post = %+v", p)
	}
	if p.PublishedAt == nil {
		t.Error("published post has no published_at")
	}
	if len(p.Tags) != 2 || p.Tags[0].Slug != "go" || p.Tags[1].Slug != "testing" {
		t.Errorf("tags = %+v", p.Tags)
	}
}

func TestAdminPostUpdate(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	id := publishPost(t, db.NewPostRepo(d), "Before", "before", "draft", "old", nil)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts/"+itoa(id), url.Values{
		"title":   {"After"},
		"slug":    {"before"},
		"status":  {"published"},
		"body_md": {"new"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	p, err := db.NewPostRepo(d).PublishedBySlug("before")
	if err != nil {
		t.Fatalf("PublishedBySlug: %v", err)
	}
	if p.Title != "After" || p.BodyMD != "new" {
		t.Errorf("post = %+v", p)
	}
}

func TestAdminPostDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	id := publishPost(t, db.NewPostRepo(d), "Gone", "gone", "draft", "body", nil)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	if _, err := db.NewPostRepo(d).ByID(id); err == nil {
		t.Error("post still exists after delete")
	}
}

func TestAdminPostPublishUnpublish(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	id := publishPost(t, db.NewPostRepo(d), "Toggle", "toggle", "draft", "body", nil)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts/"+itoa(id)+"/publish", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("publish status = %d, want 303", rec.Code)
	}
	if _, err := db.NewPostRepo(d).PublishedBySlug("toggle"); err != nil {
		t.Fatalf("post not published: %v", err)
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/posts/"+itoa(id)+"/unpublish", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("unpublish status = %d, want 303", rec.Code)
	}
	if _, err := db.NewPostRepo(d).PublishedBySlug("toggle"); err == nil {
		t.Error("post still published after unpublish")
	}
}

func TestAdminPostFormsRenderCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	for _, path := range []string{"/admin/posts/new", "/admin/posts"} {
		rec := adminRequest(t, d, store, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `name="csrf_token"`) {
			t.Errorf("%s does not render a csrf_token field", path)
		}
	}
}

func TestAdminPostRejectsEmptySlug(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts", url.Values{
		"title":  {"你好世界"},
		"slug":   {""},
		"status": {"draft"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestAdminPostRejectsDuplicateSlug(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	publishPost(t, db.NewPostRepo(d), "First", "dup-slug", "draft", "body", nil)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/posts", url.Values{
		"title":  {"Second"},
		"slug":   {"dup-slug"},
		"status": {"draft"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}
