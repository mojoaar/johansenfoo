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

func TestAPIPublicHiddenRowsExcludedPerEndpoint(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	profileRepo := db.NewProfileRepo(d)
	contentRepo := db.NewContentRepo(d)

	link, err := profileRepo.SocialLink(1)
	if err != nil {
		t.Fatalf("SocialLink: %v", err)
	}
	link.Visible = false
	if err := profileRepo.UpdateSocialLink(link); err != nil {
		t.Fatalf("UpdateSocialLink: %v", err)
	}
	exp, err := contentRepo.ExperienceItem(1)
	if err != nil {
		t.Fatalf("ExperienceItem: %v", err)
	}
	exp.Visible = false
	if err := contentRepo.UpdateExperience(exp); err != nil {
		t.Fatalf("UpdateExperience: %v", err)
	}
	skill, err := contentRepo.Skill(1)
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	skill.Visible = false
	if err := contentRepo.UpdateSkill(skill); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	rec := apiDo(t, h, http.MethodGet, "/api/v1/profile", "", nil)
	social := decodeBody[struct {
		Social []db.SocialLink `json:"social"`
	}](t, rec).Social
	if len(social) != 3 {
		t.Errorf("social = %d, want 3", len(social))
	}
	for _, l := range social {
		if l.ID == 1 {
			t.Error("hidden social link leaked through the public profile")
		}
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/experience", "", nil)
	experience := decodeBody[[]db.Experience](t, rec)
	if len(experience) != 7 {
		t.Errorf("experience = %d, want 7", len(experience))
	}
	for _, e := range experience {
		if e.ID == 1 {
			t.Error("hidden experience leaked through the public API")
		}
	}

	rec = apiDo(t, h, http.MethodGet, "/api/v1/skills", "", nil)
	skills := decodeBody[[]db.Skill](t, rec)
	if len(skills) != 29 {
		t.Errorf("skills = %d, want 29", len(skills))
	}
	for _, s := range skills {
		if s.ID == 1 {
			t.Error("hidden skill leaked through the public API")
		}
	}
}
