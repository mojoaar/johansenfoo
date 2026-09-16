package theme

import (
	"fmt"
	"sort"
)

type Theme struct {
	Slug        string
	Name        string
	Description string
	Base        map[string]string
	Light       map[string]string
	Dark        map[string]string
}

var requiredTokens = []string{
	"--bg", "--bg2", "--bg3", "--border",
	"--text", "--text-muted",
	"--accent", "--accent2", "--accent-glow", "--green",
	"--shadow",
}

var optionalTokens = []string{
	"--radius", "--font-mono", "--font-sans",
}

func known() map[string]bool {
	m := make(map[string]bool, len(requiredTokens)+len(optionalTokens))
	for _, t := range append(append([]string{}, requiredTokens...), optionalTokens...) {
		m[t] = true
	}
	return m
}

func Validate(t Theme) error {
	knownTokens := known()

	for _, section := range []struct {
		label  string
		tokens map[string]string
	}{
		{"base", t.Base},
		{"light", t.Light},
		{"dark", t.Dark},
	} {
		for name := range section.tokens {
			if !knownTokens[name] {
				return fmt.Errorf("theme %q: unknown token %q in %s", t.Slug, name, section.label)
			}
		}
	}

	for _, name := range requiredTokens {
		if t.Light[name] == "" {
			return fmt.Errorf("theme %q: missing %s in light", t.Slug, name)
		}
		if t.Dark[name] == "" {
			return fmt.Errorf("theme %q: missing %s in dark", t.Slug, name)
		}
	}
	return nil
}

func Resolve(base, mode map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(mode))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range mode {
		out[k] = v
	}
	return out
}

func Vars(tokens map[string]string) string {
	names := make([]string, 0, len(tokens))
	for name := range tokens {
		names = append(names, name)
	}
	sort.Strings(names)

	out := ""
	for i, name := range names {
		if i > 0 {
			out += ";"
		}
		out += name + ":" + tokens[name]
	}
	return out
}
