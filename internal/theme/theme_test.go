package theme

import (
	"strings"
	"testing"
)

func TestJohansenHasEveryToken(t *testing.T) {
	th := Johansen()

	if err := Validate(th); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for _, mode := range []map[string]string{th.Base, th.Light, th.Dark} {
		for name := range mode {
			if !strings.HasPrefix(name, "--") {
				t.Errorf("token %q is not a CSS custom property", name)
			}
		}
	}
}

func TestJohansenDarkValuesMatchCurrentSite(t *testing.T) {
	th := Johansen()

	want := map[string]string{
		"--bg":         "#0f1117",
		"--bg2":        "#161b27",
		"--bg3":        "#1e2433",
		"--border":     "#2a3045",
		"--text":       "#e2e8f0",
		"--text-muted": "#8892a4",
		"--accent":     "#7c6af7",
		"--accent2":    "#a78bfa",
		"--green":      "#34d399",
	}
	for name, wantVal := range want {
		if got := th.Dark[name]; got != wantVal {
			t.Errorf("dark %s = %q, want %q", name, got, wantVal)
		}
	}
}

func TestJohansenLightValuesMatchCurrentSite(t *testing.T) {
	th := Johansen()

	want := map[string]string{
		"--bg":         "#f4f6fb",
		"--bg2":        "#ffffff",
		"--bg3":        "#eef1f8",
		"--border":     "#d0d7e3",
		"--text":       "#1a1f2e",
		"--text-muted": "#5a6278",
		"--accent":     "#5b4edc",
		"--accent2":    "#7c6af7",
		"--green":      "#059669",
	}
	for name, wantVal := range want {
		if got := th.Light[name]; got != wantVal {
			t.Errorf("light %s = %q, want %q", name, got, wantVal)
		}
	}
}

func TestValidateRejectsUnknownToken(t *testing.T) {
	th := Johansen()
	th.Dark["--not-a-real-token"] = "#fff"

	if err := Validate(th); err == nil {
		t.Fatal("Validate accepted an unknown token, want error")
	}
}

func TestValidateRejectsMissingToken(t *testing.T) {
	th := Johansen()
	delete(th.Dark, "--bg")

	if err := Validate(th); err == nil {
		t.Fatal("Validate accepted a theme missing --bg in dark, want error")
	}
}

func TestValidateRejectsCSSBreakingValues(t *testing.T) {
	for _, bad := range []string{
		"a{",
		"a}",
		"a;",
		`a"}body{color:red}/*`,
	} {
		th := Johansen()
		th.Dark["--bg"] = bad

		err := Validate(th)
		if err == nil {
			t.Errorf("Validate accepted value %q, want error", bad)
			continue
		}
		if !strings.Contains(err.Error(), "--bg") {
			t.Errorf("error for %q = %q, want it to name --bg", bad, err)
		}
	}
}

func TestValidateAcceptsQuotedValues(t *testing.T) {
	th := Johansen()
	th.Base["--font-mono"] = `"JetBrains Mono", monospace`

	if err := Validate(th); err != nil {
		t.Fatalf("Validate rejected a quoted font value: %v", err)
	}
}

func TestCSSScopesByThemeAndMode(t *testing.T) {
	css := CSS(Johansen())

	if !strings.Contains(css, `[data-theme="johansen"]`) {
		t.Error("CSS does not scope to the theme slug")
	}
	if !strings.Contains(css, `[data-theme="johansen"][data-mode="dark"]`) {
		t.Error("CSS does not scope dark mode")
	}
	if !strings.Contains(css, `[data-theme="johansen"][data-mode="light"]`) {
		t.Error("CSS does not scope light mode")
	}
	if !strings.Contains(css, "--bg:#0f1117") {
		t.Error("CSS does not contain the dark --bg value")
	}
	if !strings.Contains(css, "--bg:#f4f6fb") {
		t.Error("CSS does not contain the light --bg value")
	}
	if strings.Contains(css, "--radius") && strings.Count(css, "--radius:") != 1 {
		t.Error("--radius should be emitted once, in the base block")
	}
}
