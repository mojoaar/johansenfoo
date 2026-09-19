package db

import (
	"testing"

	"github.com/mojoaar/johansenfoo/internal/theme"
)

func TestSeedThemesInsertsMissing(t *testing.T) {
	d := seeded(t)
	n, err := SeedThemes(d, theme.Seeds())
	if err != nil {
		t.Fatalf("SeedThemes: %v", err)
	}
	if n != len(theme.Seeds()) {
		t.Fatalf("inserted = %d, want %d", n, len(theme.Seeds()))
	}
	nord, err := NewThemeRepo(d).GetBySlug("nord")
	if err != nil {
		t.Fatalf("GetBySlug nord: %v", err)
	}
	if nord.Name != "Nord" || nord.TokensDark["--bg"] == "" {
		t.Fatalf("nord = %+v", nord)
	}
}

func TestSeedThemesIsIdempotentAndNonDestructive(t *testing.T) {
	d := seeded(t)
	if _, err := SeedThemes(d, theme.Seeds()); err != nil {
		t.Fatalf("SeedThemes: %v", err)
	}
	n, err := SeedThemes(d, theme.Seeds())
	if err != nil {
		t.Fatalf("SeedThemes second: %v", err)
	}
	if n != 0 {
		t.Fatalf("second insert = %d, want 0", n)
	}

	nord, err := NewThemeRepo(d).GetBySlug("nord")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	nord.Name = "Edited Nord"
	if err := NewThemeRepo(d).Update(nord); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := SeedThemes(d, theme.Seeds()); err != nil {
		t.Fatalf("SeedThemes third: %v", err)
	}
	got, err := NewThemeRepo(d).GetBySlug("nord")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.Name != "Edited Nord" {
		t.Errorf("name = %q, want Edited Nord (seed overwrote an edit)", got.Name)
	}
}
