package db

import (
	"errors"
	"testing"
	"time"
)

func newPost(t *testing.T, r *PostRepo, title, slug, status string, publishedAt *time.Time, tags []string) int64 {
	t.Helper()
	p := &Post{
		Slug:        slug,
		Title:       title,
		Summary:     "summary of " + title,
		BodyMD:      "# " + title,
		Status:      status,
		PublishedAt: publishedAt,
	}
	for _, name := range tags {
		p.Tags = append(p.Tags, Tag{Name: name, Slug: name})
	}
	id, err := r.Create(p)
	if err != nil {
		t.Fatalf("Create %s: %v", slug, err)
	}
	return id
}

func TestPostCreateAndGet(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	when := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	id := newPost(t, r, "Hello", "hello", "published", &when, []string{"go", "testing"})

	got, err := r.ByID(id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Title != "Hello" || got.Slug != "hello" || got.Status != "published" {
		t.Errorf("post = %+v", got)
	}
	if got.PublishedAt == nil || !got.PublishedAt.Equal(when) {
		t.Errorf("PublishedAt = %v, want %v", got.PublishedAt, when)
	}
	if got.CreatedAt.IsZero() || got.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt = %v, want non-zero UTC", got.CreatedAt)
	}
	if len(got.Tags) != 2 || got.Tags[0].Name != "go" || got.Tags[1].Name != "testing" {
		t.Errorf("tags = %+v", got.Tags)
	}
}

func TestPostPublishedBySlugRejectsDraft(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	id := newPost(t, r, "Draft", "draft", "draft", nil, nil)

	if _, err := r.PublishedBySlug("draft"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("PublishedBySlug draft err = %v, want ErrPostNotFound", err)
	}
	if _, err := r.ByID(id); err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if _, err := r.PublishedBySlug("draft"); err == nil {
		t.Fatal("draft exposed by PublishedBySlug")
	}
}

func TestPostPublishedOrdersNewestFirst(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	newest := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	newPost(t, r, "Old", "old", "published", &old, nil)
	newPost(t, r, "Mid", "mid", "published", &mid, nil)
	newPost(t, r, "New", "new", "published", &newest, nil)
	newPost(t, r, "Draft", "draft", "draft", nil, nil)

	posts, err := r.Published(10, 0)
	if err != nil {
		t.Fatalf("Published: %v", err)
	}
	if len(posts) != 3 {
		t.Fatalf("published = %d, want 3", len(posts))
	}
	if posts[0].Slug != "new" || posts[1].Slug != "mid" || posts[2].Slug != "old" {
		t.Errorf("order = %q, %q, %q", posts[0].Slug, posts[1].Slug, posts[2].Slug)
	}
}

func TestPostByTagFiltersPublished(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	when := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	newPost(t, r, "Go A", "go-a", "published", &when, []string{"go"})
	newPost(t, r, "Go B", "go-b", "draft", nil, []string{"go"})
	newPost(t, r, "Other", "other", "published", &when, []string{"life"})

	posts, err := r.ByTag("go", 10, 0)
	if err != nil {
		t.Fatalf("ByTag: %v", err)
	}
	if len(posts) != 1 || posts[0].Slug != "go-a" {
		t.Fatalf("ByTag go = %+v, want [go-a]", posts)
	}
	n, err := r.CountByTag("go")
	if err != nil || n != 1 {
		t.Fatalf("CountByTag = %d, err=%v", n, err)
	}
	if _, err := r.TagBySlug("life"); err != nil {
		t.Fatalf("TagBySlug life: %v", err)
	}
	if _, err := r.TagBySlug("nope"); !errors.Is(err, ErrTagNotFound) {
		t.Fatalf("TagBySlug missing err = %v, want ErrTagNotFound", err)
	}
}

func TestPostSetStatusPublishes(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	id := newPost(t, r, "Soon", "soon", "draft", nil, nil)

	if err := r.SetStatus(id, "published"); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	got, err := r.PublishedBySlug("soon")
	if err != nil {
		t.Fatalf("PublishedBySlug: %v", err)
	}
	if got.PublishedAt == nil {
		t.Fatal("PublishedAt not set on publish")
	}

	if err := r.SetStatus(id, "draft"); err != nil {
		t.Fatalf("SetStatus draft: %v", err)
	}
	if _, err := r.PublishedBySlug("soon"); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("after unpublish err = %v, want ErrPostNotFound", err)
	}
}

func TestPostUpdateReplacesTags(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	id := newPost(t, r, "Post", "post", "draft", nil, []string{"a", "b"})

	p, err := r.ByID(id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	p.Title = "Post Two"
	p.Tags = []Tag{{Name: "b", Slug: "b"}, {Name: "c", Slug: "c"}}
	if err := r.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := r.ByID(id)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Title != "Post Two" {
		t.Errorf("title = %q", got.Title)
	}
	if len(got.Tags) != 2 || got.Tags[0].Slug != "b" || got.Tags[1].Slug != "c" {
		t.Errorf("tags = %+v", got.Tags)
	}
}

func TestPostDeleteRemovesTagLinks(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	id := newPost(t, r, "Gone", "gone", "draft", nil, []string{"x", "y"})

	if err := r.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.ByID(id); !errors.Is(err, ErrPostNotFound) {
		t.Fatalf("after delete err = %v, want ErrPostNotFound", err)
	}
	var links int
	if err := d.QueryRow(`SELECT COUNT(*) FROM post_tag WHERE post_id = ?`, id).Scan(&links); err != nil {
		t.Fatalf("count post_tag: %v", err)
	}
	if links != 0 {
		t.Errorf("post_tag rows = %d, want 0", links)
	}
}

func TestPostAllIncludesDrafts(t *testing.T) {
	d := seeded(t)
	r := NewPostRepo(d)
	when := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	newPost(t, r, "Pub", "pub", "published", &when, nil)
	newPost(t, r, "Dr", "dr", "draft", nil, nil)

	all, err := r.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("All = %d, want 2", len(all))
	}
	tags, err := r.Tags()
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if tags != nil && len(tags) != 0 {
		t.Errorf("Tags = %+v, want empty", tags)
	}
}
