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

	if !strings.Contains(rec.Body.String(), "--bg:#0f1117") {
		t.Error("landing page does not inline the active theme")
	}
}
