package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestMeReturnsJSONFromDatabase(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}

	var got struct {
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name        string `json:"name"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if got.Name != "Morten Johansen" || got.Handle != "mojoaar" || got.Location != "Denmark" {
		t.Errorf("identity fields wrong: %+v", got)
	}
	if got.DOB != "1980-08-13" {
		t.Errorf("dob = %q", got.DOB)
	}
	if len(got.Skills) != 30 {
		t.Errorf("got %d skills, want 30", len(got.Skills))
	}
	if len(got.Projects) != 9 {
		t.Errorf("got %d projects, want 9", len(got.Projects))
	}
	if got.Projects[3].Name != "homelab" || got.Projects[3].URL != "" {
		t.Errorf("homelab project = %+v, want empty url", got.Projects[3])
	}
	if got.Social["github"] != "https://github.com/mojoaar" {
		t.Errorf("social github = %q", got.Social["github"])
	}
}

func TestMeParityWithLegacyFileStructureOnly(t *testing.T) {
	const legacyPath = "/Users/mojoaar/Development/johansen_landing/me"
	raw, err := os.ReadFile(legacyPath)
	if err != nil {
		t.Skipf("legacy me file unavailable, skipping structural parity check: %v", err)
	}

	var old struct {
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name string  `json:"name"`
			URL  *string `json:"url"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(raw, &old); err != nil {
		t.Fatalf("parse legacy me: %v", err)
	}

	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var got struct {
		Name     string            `json:"name"`
		Handle   string            `json:"handle"`
		Location string            `json:"location"`
		DOB      string            `json:"dob"`
		Skills   []string          `json:"skills"`
		Social   map[string]string `json:"social"`
		Projects []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}

	if got.Name != old.Name || got.Handle != old.Handle || got.Location != old.Location || got.DOB != old.DOB {
		t.Errorf("profile fields differ: got %q/%q/%q/%q, want %q/%q/%q/%q",
			got.Name, got.Handle, got.Location, got.DOB,
			old.Name, old.Handle, old.Location, old.DOB)
	}
	if !slices.Equal(got.Skills, old.Skills) {
		t.Errorf("skills differ: got %d entries, want %d", len(got.Skills), len(old.Skills))
	}
	if len(got.Social) != len(old.Social) {
		t.Errorf("social has %d keys, want %d", len(got.Social), len(old.Social))
	}
	for k, v := range old.Social {
		if got.Social[k] != v {
			t.Errorf("social[%q] = %q, want %q", k, got.Social[k], v)
		}
	}
	if len(got.Projects) != len(old.Projects) {
		t.Fatalf("got %d projects, want %d", len(got.Projects), len(old.Projects))
	}
	for i, p := range old.Projects {
		wantURL := ""
		if p.URL != nil {
			wantURL = *p.URL
		}
		if got.Projects[i].Name != p.Name {
			t.Errorf("project[%d].name = %q, want %q", i, got.Projects[i].Name, p.Name)
		}
		if got.Projects[i].URL != wantURL {
			t.Errorf("project[%d].url = %q, want %q", i, got.Projects[i].URL, wantURL)
		}
	}
}

func TestRobotsTxt(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "User-agent: *") {
		t.Error("robots.txt is missing the user-agent line")
	}
	if !strings.Contains(body, "Sitemap: https://johansen.foo/sitemap.xml") {
		t.Error("robots.txt is missing the sitemap line")
	}
}

func TestSitemapXML(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, `<?xml`) {
		t.Error("sitemap does not start with an XML declaration")
	}
	if !strings.Contains(body, "<loc>https://johansen.foo/</loc>") {
		t.Error("sitemap is missing the home URL")
	}
	if strings.Contains(body, "/posts") {
		t.Error("phase 1 has no posts and the sitemap must not advertise them")
	}
}

func TestMetaPrefersSettingsOverFallback(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "<title>Morten Johansen | johansen.foo</title>") {
		t.Error("title does not come from settings")
	}
	if !strings.Contains(body, `<link rel="canonical" href="https://johansen.foo/" />`) {
		t.Error("canonical URL is wrong")
	}
	if !strings.Contains(body, `content="index, follow"`) {
		t.Error("robots meta is wrong")
	}
}
