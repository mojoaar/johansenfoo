package db

import (
	"errors"
	"testing"
)

func TestThemeListAndCreate(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)

	list, err := r.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, th := range list {
		if th.Slug == "johansen" {
			found = true
		}
	}
	if !found {
		t.Fatal("seeded johansen theme missing from List")
	}

	id, err := r.Create(&Theme{
		Slug:        "custom",
		Name:        "Custom",
		Description: "d",
		TokensBase:  map[string]string{"--accent": "#111"},
		TokensLight: map[string]string{"--accent": "#222"},
		TokensDark:  map[string]string{"--accent": "#333"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := r.GetBySlug("custom")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.ID != id || got.TokensDark["--accent"] != "#333" {
		t.Fatalf("theme = %+v", got)
	}
}

func TestThemeUpdateMergesTokens(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)
	id, err := r.Create(&Theme{
		Slug:        "merge",
		Name:        "Merge",
		TokensLight: map[string]string{"--accent": "#111"},
		TokensDark:  map[string]string{"--bg": "#000"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := r.Update(&Theme{
		ID:          id,
		Slug:        "merge",
		Name:        "Merge Two",
		TokensLight: map[string]string{"--green": "#0f0"},
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := r.GetBySlug("merge")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.Name != "Merge Two" {
		t.Errorf("name = %q", got.Name)
	}
	if got.TokensLight["--accent"] != "#111" || got.TokensLight["--green"] != "#0f0" {
		t.Errorf("light tokens not merged: %+v", got.TokensLight)
	}
	if got.TokensDark["--bg"] != "#000" {
		t.Errorf("untouched section lost: %+v", got.TokensDark)
	}
}

func TestThemeDeleteGuardRails(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)

	base, err := r.GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if err := r.Delete(base.ID); !errors.Is(err, ErrThemeProtected) {
		t.Fatalf("deleting johansen err = %v, want ErrThemeProtected", err)
	}

	id, err := r.Create(&Theme{Slug: "active-one", Name: "Active One"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.SetActive("active-one"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := r.Delete(id); !errors.Is(err, ErrThemeProtected) {
		t.Fatalf("deleting active theme err = %v, want ErrThemeProtected", err)
	}

	if err := r.SetActive("johansen"); err != nil {
		t.Fatalf("SetActive johansen: %v", err)
	}
	if err := r.Delete(id); err != nil {
		t.Fatalf("Delete inactive theme: %v", err)
	}
}

func TestThemeSetActive(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)

	if err := r.SetActive("missing"); !errors.Is(err, ErrThemeNotFound) {
		t.Fatalf("SetActive missing err = %v, want ErrThemeNotFound", err)
	}
	if slug, err := r.ActiveSlug(); err != nil || slug != "johansen" {
		t.Fatalf("ActiveSlug = %q, err=%v", slug, err)
	}
}
