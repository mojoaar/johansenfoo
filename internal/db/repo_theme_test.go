package db

import (
	"database/sql"
	"errors"
	"testing"
)

func validTheme(t *testing.T, d *sql.DB, slug, name string) *Theme {
	t.Helper()
	base, err := NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug johansen: %v", err)
	}
	return &Theme{
		Slug:        slug,
		Name:        name,
		Description: "test",
		TokensBase:  base.TokensBase,
		TokensLight: base.TokensLight,
		TokensDark:  base.TokensDark,
	}
}

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

	th := validTheme(t, d, "custom", "Custom")
	th.TokensDark["--accent"] = "#333"
	id, err := r.Create(th)
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
	th := validTheme(t, d, "merge", "Merge")
	id, err := r.Create(th)
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
	if got.TokensLight["--accent"] == "" || got.TokensLight["--green"] != "#0f0" {
		t.Errorf("light tokens not merged: %+v", got.TokensLight)
	}
}

func TestThemeRepoRejectsInvalidTokens(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)

	bad := validTheme(t, d, "bad-token", "Bad")
	bad.TokensDark["--nope"] = "x"
	if _, err := r.Create(bad); err == nil {
		t.Error("Create accepted an unknown token")
	}

	inject := validTheme(t, d, "inject", "Inject")
	inject.TokensDark["--accent"] = "</style>"
	if _, err := r.Create(inject); err == nil {
		t.Error("Create accepted a CSS-breaking value")
	}

	valid := validTheme(t, d, "valid-theme", "Valid")
	id, err := r.Create(valid)
	if err != nil {
		t.Fatalf("Create valid: %v", err)
	}
	valid.ID = id
	valid.TokensDark["--accent"] = "a<b"
	if err := r.Update(valid); err == nil {
		t.Error("Update accepted a CSS-breaking value")
	}
}

func TestThemeRepoUpdateRefusesProtectedRename(t *testing.T) {
	d := seeded(t)
	r := NewThemeRepo(d)
	base, err := r.GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	renamed := *base
	renamed.Slug = "renamed-base"
	if err := r.Update(&renamed); !errors.Is(err, ErrThemeProtected) {
		t.Fatalf("renaming johansen err = %v, want ErrThemeProtected", err)
	}
	if _, err := r.GetBySlug("johansen"); err != nil {
		t.Fatalf("johansen missing after refused rename: %v", err)
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

	id, err := r.Create(validTheme(t, d, "active-one", "Active One"))
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
