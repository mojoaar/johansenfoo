package theme

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

func validSlug(s string) bool {
	return len(s) > 0 && len(s) <= 64 && slugPattern.MatchString(s)
}

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
	"--card", "--card-foreground", "--popover", "--popover-foreground",
	"--primary", "--primary-foreground", "--secondary", "--secondary-foreground",
	"--muted", "--muted-foreground", "--accent-foreground",
	"--destructive", "--destructive-foreground", "--input", "--ring",
	"--font-size-base", "--line-height", "--letter-spacing",
	"--font-weight-normal", "--font-weight-bold",
	"--border-width", "--shadow-sm", "--shadow-lg",
	"--code-bg", "--code-text", "--code-keyword", "--code-string",
	"--code-comment", "--code-function", "--code-number", "--code-operator",
}

func known() map[string]bool {
	m := make(map[string]bool, len(requiredTokens)+len(optionalTokens))
	for _, t := range append(append([]string{}, requiredTokens...), optionalTokens...) {
		m[t] = true
	}
	return m
}

func Validate(t Theme) error {
	if !validSlug(t.Slug) {
		return fmt.Errorf("theme slug %q is invalid", t.Slug)
	}
	knownTokens := known()

	for _, section := range []struct {
		label  string
		tokens map[string]string
	}{
		{"base", t.Base},
		{"light", t.Light},
		{"dark", t.Dark},
	} {
		for name, value := range section.tokens {
			if !knownTokens[name] {
				return fmt.Errorf("theme %q: unknown token %q in %s", t.Slug, name, section.label)
			}
			if strings.ContainsAny(value, "{};<>") {
				return fmt.Errorf("theme %q: token %q in %s contains a CSS-breaking character", t.Slug, name, section.label)
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
