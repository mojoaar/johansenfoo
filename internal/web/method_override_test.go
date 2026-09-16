package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestMethodOverrideRoutesToAdminHandlers(t *testing.T) {
	cases := []struct {
		name   string
		method string
		seed   func(t *testing.T, d *sql.DB) (path string, form url.Values)
		check  func(t *testing.T, d *sql.DB)
	}{
		{
			name:   "PUT reaches the social update handler",
			method: http.MethodPut,
			seed: func(t *testing.T, d *sql.DB) (string, url.Values) {
				t.Helper()
				id, err := db.NewProfileRepo(d).CreateSocialLink(&db.SocialLink{
					Platform: "override", URL: "https://old.example.com", Label: "Old", Sort: 42, Visible: true,
				})
				if err != nil {
					t.Fatalf("CreateSocialLink: %v", err)
				}
				form := url.Values{
					"platform": {"override"}, "url": {"https://new.example.com"},
					"label": {"Updated"}, "sort": {"42"}, "visible": {"1"},
				}
				return "/admin/social/" + itoa(id), form
			},
			check: func(t *testing.T, d *sql.DB) {
				t.Helper()
				links, err := db.NewProfileRepo(d).SocialLinks()
				if err != nil {
					t.Fatalf("SocialLinks: %v", err)
				}
				for _, l := range links {
					if l.Platform == "override" {
						if l.Label != "Updated" {
							t.Errorf("social label = %q, want %q", l.Label, "Updated")
						}
						return
					}
				}
				t.Error("updated social link is missing")
			},
		},
		{
			name:   "PUT reaches the project update handler",
			method: http.MethodPut,
			seed: func(t *testing.T, d *sql.DB) (string, url.Values) {
				t.Helper()
				id, err := db.NewContentRepo(d).CreateProject(&db.Project{
					Name: "Override", Description: "old", Sort: 42, Visible: true,
				})
				if err != nil {
					t.Fatalf("CreateProject: %v", err)
				}
				form := url.Values{
					"name": {"Renamed"}, "description": {"new"}, "sort": {"42"}, "visible": {"1"},
				}
				return "/admin/projects/" + itoa(id), form
			},
			check: func(t *testing.T, d *sql.DB) {
				t.Helper()
				projects, err := db.NewContentRepo(d).Projects()
				if err != nil {
					t.Fatalf("Projects: %v", err)
				}
				for _, p := range projects {
					if p.Name == "Renamed" {
						return
					}
				}
				t.Error("updated project is missing")
			},
		},
		{
			name:   "DELETE reaches the social delete handler",
			method: http.MethodDelete,
			seed: func(t *testing.T, d *sql.DB) (string, url.Values) {
				t.Helper()
				id, err := db.NewProfileRepo(d).CreateSocialLink(&db.SocialLink{
					Platform: "override", URL: "https://delete.example.com", Label: "Delete", Sort: 42, Visible: true,
				})
				if err != nil {
					t.Fatalf("CreateSocialLink: %v", err)
				}
				return "/admin/social/" + itoa(id) + "/delete", url.Values{}
			},
			check: func(t *testing.T, d *sql.DB) {
				t.Helper()
				links, err := db.NewProfileRepo(d).SocialLinks()
				if err != nil {
					t.Fatalf("SocialLinks: %v", err)
				}
				for _, l := range links {
					if l.Platform == "override" {
						t.Fatal("social link survived a _method=DELETE override")
					}
				}
			},
		},
		{
			name:   "DELETE reaches the project delete handler",
			method: http.MethodDelete,
			seed: func(t *testing.T, d *sql.DB) (string, url.Values) {
				t.Helper()
				id, err := db.NewContentRepo(d).CreateProject(&db.Project{
					Name: "Override delete", Description: "delete", Sort: 42, Visible: true,
				})
				if err != nil {
					t.Fatalf("CreateProject: %v", err)
				}
				return "/admin/projects/" + itoa(id) + "/delete", url.Values{}
			},
			check: func(t *testing.T, d *sql.DB) {
				t.Helper()
				projects, err := db.NewContentRepo(d).Projects()
				if err != nil {
					t.Fatalf("Projects: %v", err)
				}
				for _, p := range projects {
					if p.Name == "Override delete" {
						t.Fatal("project survived a _method=DELETE override")
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, _ := NewContentStore(d)
			h := loggedInHandler(t, d, store)
			path, form := tc.seed(t, d)

			get := httptest.NewRequest(http.MethodGet, "/admin", nil)
			get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			getRec := httptest.NewRecorder()
			h.ServeHTTP(getRec, get)
			token := findCookie(t, getRec.Result().Cookies(), csrfCookieName).Value

			form.Set("csrf_token", token)
			form.Set("_method", tc.method)

			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code == http.StatusMethodNotAllowed {
				t.Fatalf("POST %s with _method=%s returned 405; the override did not reach a handler", path, tc.method)
			}
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("POST %s with _method=%s returned %d, want %d; body=%s",
					path, tc.method, rec.Code, http.StatusSeeOther, rec.Body.String())
			}
			tc.check(t, d)
		})
	}
}
