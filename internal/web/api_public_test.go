package web

import (
	"net/http"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAPIPublicProfileIncludesSocial(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/profile", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	got := decodeBody[struct {
		Name   string          `json:"name"`
		Social []db.SocialLink `json:"social"`
	}](t, rec)
	if got.Name != "Morten Johansen" {
		t.Errorf("name = %q, want Morten Johansen", got.Name)
	}
	if len(got.Social) != 4 {
		t.Errorf("social = %d, want 4", len(got.Social))
	}
}

func TestAPIPublicListsExcludeHiddenRows(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)

	hidden := "hidden-proj"
	if _, err := db.NewContentRepo(d).CreateProject(&db.Project{Name: hidden, Visible: false}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec := apiDo(t, h, http.MethodGet, "/api/v1/projects", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	projects := decodeBody[[]db.Project](t, rec)
	for _, p := range projects {
		if p.Name == hidden {
			t.Fatal("hidden project leaked through the public API")
		}
	}
	if len(projects) != 9 {
		t.Errorf("projects = %d, want 9", len(projects))
	}
}

func TestAPIPublicThemeIsResolved(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/theme", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	got := decodeBody[struct {
		Slug  string            `json:"slug"`
		Light map[string]string `json:"light"`
		Dark  map[string]string `json:"dark"`
	}](t, rec)
	if got.Slug != "johansen" {
		t.Errorf("slug = %q, want johansen", got.Slug)
	}
	if got.Light["--bg"] == "" || got.Dark["--bg"] == "" {
		t.Errorf("theme tokens not resolved: light=%q dark=%q", got.Light["--bg"], got.Dark["--bg"])
	}
}
