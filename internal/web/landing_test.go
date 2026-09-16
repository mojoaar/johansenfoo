package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLandingRendersContentFromDatabase(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()

	for _, want := range []string{
		"Morten Johansen",
		"// enterprise it leader &amp; open-source tinkerer",
		"atlascmdb",
		"mindmatrix",
		"self-hosted // private",
		"Jydske Dragonregiment",
		"ITSM",
		"Presenting",
		`href="https://jysk.com"`,
		`data-theme="johansen"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("landing page is missing %q", want)
		}
	}

	if strings.Contains(body, "cdn.jsdelivr.net") {
		t.Error("landing page still loads Lucide from a CDN")
	}
	if strings.Contains(body, "font-awesome") {
		t.Error("landing page still loads Font Awesome")
	}
	if strings.Contains(body, "data-lucide=") {
		t.Error("landing page still contains data-lucide placeholders")
	}
}

func TestLandingInlineThemeIsPresent(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()

	for _, tc := range []struct {
		name string
		want string
	}{
		{"dark background token", "--bg:#0f1117"},
		{"base theme scope", `[data-theme="johansen"]{`},
		{"dark mode scope", `[data-theme="johansen"][data-mode="dark"]{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(body, tc.want) {
				t.Errorf("landing page is missing %q", tc.want)
			}
		})
	}

	head := headSection(t, body)
	if !strings.Contains(head, "<style>") {
		t.Error("inlined theme block is not inside <head>")
	}
	if !strings.Contains(head, "--bg:#0f1117") {
		t.Error("inlined theme block in <head> does not carry the dark --bg token")
	}
}

func TestLandingRendersInlineSvgIcons(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()

	for _, tc := range []struct {
		name string
		want []string
	}{
		{"social github icon", []string{`aria-label="GitHub"`, `class="social-github"`, `<svg`, `class=""`}},
		{"project icon", []string{`class="project-icon"`, `<svg`, `aria-hidden="true"`, `focusable="false"`}},
		{"project link icon", []string{`class="project-link-icon"`, `<svg`}},
		{"experience icon", []string{`class="tl-icon"`, `<svg`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			markup := markupAfter(t, body, tc.want[0])
			for _, want := range tc.want[1:] {
				if !strings.Contains(markup, want) {
					t.Errorf("markup at %q is missing %q; got:\n%s", tc.want[0], want, markup)
				}
			}
		})
	}

	if n := strings.Count(body, "<svg"); n == 0 {
		t.Fatal("landing page contains no inline <svg> markup; the icon funcmap is not wired")
	}
	if strings.Contains(body, "<i data-lucide") || strings.Contains(body, "<i ") {
		t.Error("landing page contains <i> icon placeholders instead of inline SVG")
	}
}

func TestLandingThemeAndModeAreSeparateAttributes(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()

	html := openTag(t, body, "<html")

	if !strings.Contains(html, `data-theme="johansen"`) {
		t.Errorf("<html> tag is missing data-theme=\"johansen\"; got:\n%s", html)
	}
	if !strings.Contains(html, `data-mode="dark"`) {
		t.Errorf("<html> tag is missing data-mode=\"dark\"; got:\n%s", html)
	}
	if strings.Contains(html, `data-theme="johansen" data-mode`) &&
		!strings.Contains(html, `data-theme="johansen" data-mode="dark"`) {
		t.Error("<html> tag conflates data-theme and data-mode")
	}
}

func TestLandingIncludesUmamiAnalytics(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	body := rec.Body.String()

	head := headSection(t, body)
	if !strings.Contains(head, `src="https://umami.johansen.foo/script.js"`) {
		t.Error("landing page <head> is missing the umami script src")
	}
	if !strings.Contains(head, `data-website-id="427f3677-4613-47a2-8128-d228120321e2"`) {
		t.Errorf("landing page <head> is missing the umami website id")
	}
}

func markupAfter(t *testing.T, body, anchor string) string {
	t.Helper()

	i := strings.Index(body, anchor)
	if i < 0 {
		t.Fatalf("anchored markup %q not found in landing page", anchor)
	}
	rest := body[i:]
	if end := strings.Index(rest, "</a>"); end >= 0 {
		return rest[:end]
	}
	if end := strings.Index(rest, "</div>"); end >= 0 {
		return rest[:end]
	}
	if end := strings.Index(rest, "</span>"); end >= 0 {
		return rest[:end]
	}
	return rest
}

func headSection(t *testing.T, body string) string {
	t.Helper()

	start := strings.Index(body, "<head>")
	end := strings.Index(body, "</head>")
	if start < 0 || end < start {
		t.Fatalf("landing page has no <head> section")
	}
	return body[start:end]
}

func openTag(t *testing.T, body, name string) string {
	t.Helper()

	start := strings.Index(body, name)
	if start < 0 {
		t.Fatalf("landing page has no %s element", name)
	}
	end := strings.IndexByte(body[start:], '>')
	if end < 0 {
		t.Fatalf("landing page has an unterminated %s element", name)
	}
	return body[start : start+end+1]
}
