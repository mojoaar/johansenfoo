package icons

import (
	"regexp"
	"strings"
	"testing"
)

var fixedDimensionRE = regexp.MustCompile(`\s(width|height)="`)

func TestEveryIconNameResolves(t *testing.T) {
	names := []string{
		"database", "message-square", "globe", "server", "mail", "flame",
		"kanban", "shield-check", "brain", "briefcase", "square-terminal",
		"shield", "arrow-up-right-from-square",
		"bluesky", "linkedin", "mastodon", "github",
	}
	for _, name := range names {
		if !Exists(name) {
			t.Errorf("icon %q is not vendored", name)
			continue
		}
		svg := string(Inline(name, "project-icon"))
		if !strings.HasPrefix(svg, "<svg") {
			t.Errorf("icon %q does not start with <svg: %q", name, svg)
		}
		if !strings.Contains(svg, `class="project-icon"`) {
			t.Errorf("icon %q is missing the supplied class", name)
		}
		root := svg[:strings.IndexByte(svg, '>')+1]
		if fixedDimensionRE.MatchString(root) {
			t.Errorf("icon %q still carries fixed dimensions on its root tag", name)
		}
	}
}

func TestInnerGeometryIsPreserved(t *testing.T) {
	cases := []struct {
		name  string
		attrs []string
	}{
		{"briefcase", []string{`width="20"`, `height="14"`}},
		{"mail", []string{`width="20"`, `height="16"`}},
		{"server", []string{`width="20"`, `height="8"`}},
		{"square-terminal", []string{`width="18"`, `height="18"`}},
	}
	for _, tc := range cases {
		svg := string(Inline(tc.name, "project-icon"))
		end := strings.IndexByte(svg, '>')
		if end < 0 {
			t.Fatalf("icon %q has no root tag", tc.name)
		}
		inner := svg[end+1:]
		for _, attr := range tc.attrs {
			if !strings.Contains(inner, attr) {
				t.Errorf("icon %q lost inner geometry %s: %q", tc.name, attr, inner)
			}
		}
	}
}

func TestUnknownIconReturnsEmpty(t *testing.T) {
	if got := Inline("no-such-icon", "x"); got != "" {
		t.Errorf("Inline(unknown) = %q, want empty", got)
	}
}

func TestInlineSetsAccessibilityAttributes(t *testing.T) {
	svg := string(Inline("globe", "project-icon"))
	if !strings.Contains(svg, `aria-hidden="true"`) {
		t.Error("icon is missing aria-hidden")
	}
	if !strings.Contains(svg, `focusable="false"`) {
		t.Error("icon is missing focusable=false")
	}
}

func TestFillSemantics(t *testing.T) {
	outline := string(Inline("globe", "project-icon"))
	if !strings.Contains(outline, `fill="none"`) {
		t.Error("lucide outline icon lost fill=none")
	}
	if strings.Contains(outline, `fill="currentColor"`) {
		t.Error("lucide outline icon should not be forced to currentColor")
	}
	brand := string(Inline("github", "social-icon"))
	if !strings.Contains(brand, `fill="currentColor"`) {
		t.Error("brand icon should inherit currentColor")
	}
}
