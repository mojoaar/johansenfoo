package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAPIPostsListsPublishedOnly(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Live", "live", "published", "# Body", nil)
	publishPost(t, repo, "Hidden", "hidden", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/posts", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[apiPostsResponse](t, rec)
	if got.Total != 1 || len(got.Posts) != 1 || got.Posts[0].Slug != "live" {
		t.Fatalf("response = %+v", got)
	}
	if got.Posts[0].HTML == "" {
		t.Error("post html not rendered")
	}
	if got.PerPage != 10 {
		t.Errorf("per_page = %d, want 10", got.PerPage)
	}
}

func TestAPIPostsPaginateAndFilter(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	for i := 0; i < 12; i++ {
		slug := "post-" + string(rune('a'+i))
		publishPost(t, repo, "Post "+slug, slug, "published", "body", nil)
	}
	publishPost(t, repo, "Tagged", "tagged", "published", "body", []string{"go"})

	rec := apiDo(t, h, http.MethodGet, "/api/v1/posts?page=2", "", nil)
	got := decodeBody[apiPostsResponse](t, rec)
	if got.Page != 2 || len(got.Posts) != 3 || got.Total != 13 {
		t.Fatalf("page 2 = %+v", got)
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/posts?tag=go", "", nil)
	got = decodeBody[apiPostsResponse](t, rec)
	if got.Total != 1 || len(got.Posts) != 1 || got.Posts[0].Slug != "tagged" {
		t.Fatalf("tag filter = %+v", got)
	}
}

func TestAPIPostBySlug(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPostRepo(d)
	publishPost(t, repo, "Live", "live", "published", "```go\nfunc main() {}\n```", nil)
	publishPost(t, repo, "Draft", "draft", "draft", "body", nil)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/posts/live", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[apiPost](t, rec)
	if got.BodyMD == "" || !strings.Contains(got.HTML, `class="chroma"`) {
		t.Errorf("post = %+v", got)
	}

	if rec := apiDo(t, h, http.MethodGet, "/api/v1/posts/draft", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("draft status = %d, want 404", rec.Code)
	}
}

func TestAPIPostsDisabled(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	publishPost(t, db.NewPostRepo(d), "Live", "live", "published", "body", nil)
	if err := db.NewSettingsRepo(d).Set("posts_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	for _, path := range []string{"/api/v1/posts", "/api/v1/posts/live"} {
		if rec := apiDo(t, h, http.MethodGet, path, "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, rec.Code)
		}
	}
}

func TestAPIAdminPostsRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/posts", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminPostsCRUD(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodPost, "/api/v1/admin/posts",
		`{"title":"API Post","slug":"api-post","summary":"s","body_md":"# x","status":"published","tags":[{"name":"go","slug":"go"}]}`, withSession)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	created := decodeBody[db.Post](t, rec)
	if created.ID == 0 || created.PublishedAt == nil || len(created.Tags) != 1 {
		t.Fatalf("created = %+v", created)
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/admin/posts", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if list := decodeBody[[]db.Post](t, rec); len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}

	rec = apiDo(t, h, http.MethodPut, "/api/v1/admin/posts/"+itoa(created.ID),
		`{"title":"API Post Two","slug":"api-post","body_md":"# y","status":"published"}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if updated := decodeBody[db.Post](t, rec); updated.Title != "API Post Two" {
		t.Errorf("updated = %+v", updated)
	}

	rec = apiDo(t, h, http.MethodPost, "/api/v1/admin/posts/"+itoa(created.ID)+"/unpublish", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("unpublish status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if _, err := db.NewPostRepo(d).PublishedBySlug("api-post"); err == nil {
		t.Error("post still published after unpublish")
	}

	rec = apiDo(t, h, http.MethodDelete, "/api/v1/admin/posts/"+itoa(created.ID), "", withSession)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
}
