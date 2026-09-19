package markdown

import (
	"strings"
	"testing"
)

func TestRenderHighlightsCodeWithClasses(t *testing.T) {
	out := string(Render("# Title\n\n```go\nfunc main() {}\n```\n"))
	if !strings.Contains(out, `class="chroma"`) {
		t.Errorf("output has no chroma wrapper: %s", out)
	}
	if !strings.Contains(out, `<span class="k`) {
		t.Errorf("output has no highlighted keyword span: %s", out)
	}
	if strings.Contains(out, `style="`) {
		t.Errorf("output uses inline styles instead of classes: %s", out)
	}
}

func TestRenderEscapesRawHTML(t *testing.T) {
	out := string(Render("<script>alert(1)</script>"))
	if strings.Contains(strings.ToLower(out), "<script") {
		t.Errorf("raw script element survived rendering: %s", out)
	}
}

func TestRenderSanitisesJavascriptURL(t *testing.T) {
	out := string(Render("[x](javascript:alert(1))"))
	if strings.Contains(strings.ToLower(out), `href="javascript:`) {
		t.Errorf("javascript: URL survived rendering: %s", out)
	}
}

func TestRenderGFM(t *testing.T) {
	out := string(Render("| a | b |\n| - | - |\n| 1 | 2 |\n\n~~gone~~\n"))
	if !strings.Contains(out, "<table") {
		t.Errorf("GFM table not rendered: %s", out)
	}
	if !strings.Contains(out, "<del>") {
		t.Errorf("GFM strikethrough not rendered: %s", out)
	}
}
