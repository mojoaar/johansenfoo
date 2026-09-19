package theme

import (
	"testing"

	catppuccingo "github.com/catppuccin/go"
)

func TestSeedsAreValidAndUnique(t *testing.T) {
	seeds := Seeds()
	if len(seeds) != 10 {
		t.Fatalf("seeds = %d, want 10", len(seeds))
	}
	seen := map[string]bool{}
	for _, s := range seeds {
		if seen[s.Slug] {
			t.Errorf("duplicate slug %q", s.Slug)
		}
		seen[s.Slug] = true
		if err := Validate(s); err != nil {
			t.Errorf("seed %q invalid: %v", s.Slug, err)
		}
	}
	for _, want := range []string{
		"catppuccin-latte", "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha",
		"nord", "rose-pine", "tokyo-night", "gruvbox", "everforest", "solarized",
	} {
		if !seen[want] {
			t.Errorf("missing seed %q", want)
		}
	}
}

func TestMochaAccentMatchesCatppuccin(t *testing.T) {
	var mocha Theme
	for _, s := range Seeds() {
		if s.Slug == "catppuccin-mocha" {
			mocha = s
		}
	}
	if mocha.Dark["--accent"] != catppuccingo.Mocha.Mauve().Hex {
		t.Errorf("mocha accent = %q, want %q", mocha.Dark["--accent"], catppuccingo.Mocha.Mauve().Hex)
	}
	if mocha.Light["--bg"] != catppuccingo.Latte.Base().Hex {
		t.Errorf("mocha light bg = %q, want latte base", mocha.Light["--bg"])
	}
}
