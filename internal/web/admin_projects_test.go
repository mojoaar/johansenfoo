package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminProjectsListShowsSeededProjects(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/projects", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"atlascmdb", "mindmatrix", "/admin/projects/new"} {
		if !strings.Contains(body, want) {
			t.Errorf("project list missing %q", want)
		}
	}
}

func TestAdminProjectCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/projects", url.Values{
		"name": {"zzz"}, "url": {"https://example.com"}, "description": {"desc"},
		"icon": {"globe"}, "is_link": {"1"}, "url_label": {"example.com"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if len(store.Current().Projects) != 10 {
		t.Fatalf("snapshot projects = %d, want 10", len(store.Current().Projects))
	}

	var id int64
	for _, p := range store.Current().Projects {
		if p.Name == "zzz" {
			id = p.ID
		}
	}
	if id == 0 {
		t.Fatal("new project not found in the snapshot")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/projects/"+itoa(id), url.Values{
		"name": {"yyy"}, "url": {""}, "description": {"desc2"},
		"icon": {"globe"}, "url_label": {""}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	p, err := db.NewContentRepo(d).Project(id)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Name != "yyy" || p.IsLink {
		t.Errorf("after update = %+v", p)
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/projects/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Projects) != 9 {
		t.Errorf("snapshot projects = %d, want 9", len(store.Current().Projects))
	}
}

func TestAdminProjectEditPrefillsValues(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/projects/1", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `value="atlascmdb"`) {
		t.Error("edit form does not prefill the project name")
	}
}

func projectFromSnapshot(t *testing.T, store *ContentStore, id int64) db.Project {
	t.Helper()
	for _, p := range store.Current().Projects {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("project %d is missing from the content snapshot", id)
	return db.Project{}
}

func TestAdminProjectUpdateRefreshesContentSnapshot(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	id, err := db.NewContentRepo(d).CreateProject(&db.Project{
		Name: "snapshot", URL: "https://old.example.com", Description: "old", Icon: "globe",
		IsLink: true, URLLabel: "old.example.com", Sort: 99, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := projectFromSnapshot(t, store, id); got.URL != "https://old.example.com" {
		t.Fatalf("precondition: snapshot URL = %q, want %q", got.URL, "https://old.example.com")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/projects/"+itoa(id), url.Values{
		"name": {"snapshot-renamed"}, "url": {"https://new.example.com"}, "description": {"new"},
		"icon": {"globe"}, "url_label": {"new.example.com"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	got := projectFromSnapshot(t, store, id)
	if got.Name != "snapshot-renamed" {
		t.Errorf("snapshot Name = %q, want %q", got.Name, "snapshot-renamed")
	}
	if got.URL != "https://new.example.com" {
		t.Errorf("snapshot URL = %q, want %q", got.URL, "https://new.example.com")
	}
}

func TestAdminProjectHiddenRowsStayOutOfPublicSite(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	id, err := db.NewContentRepo(d).CreateProject(&db.Project{
		Name: "zzzhidden", URL: "https://hidden.example.com", Description: "hidden description",
		Icon: "globe", IsLink: true, URLLabel: "hidden.example.com", Sort: 50, Visible: false,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	found := false
	for _, p := range store.Current().Projects {
		if p.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("hidden project is missing from the admin content snapshot")
	}

	adminGet := httptest.NewRequest(http.MethodGet, "/admin/projects", nil)
	adminGet.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	adminRec := httptest.NewRecorder()
	h.ServeHTTP(adminRec, adminGet)
	if !strings.Contains(adminRec.Body.String(), "zzzhidden") {
		t.Error("admin project list does not show the hidden row for re-enabling")
	}

	landing := httptest.NewRequest(http.MethodGet, "/", nil)
	landingRec := httptest.NewRecorder()
	h.ServeHTTP(landingRec, landing)
	if strings.Contains(landingRec.Body.String(), "zzzhidden") {
		t.Error("hidden project leaked onto /")
	}

	me := httptest.NewRequest(http.MethodGet, "/me", nil)
	meRec := httptest.NewRecorder()
	h.ServeHTTP(meRec, me)
	if strings.Contains(meRec.Body.String(), "zzzhidden") {
		t.Error("hidden project leaked into /me")
	}

	ld := landingRec.Body.String()
	const marker = `<script type="application/ld+json">`
	i := strings.Index(ld, marker)
	if i < 0 {
		t.Fatal("landing page has no JSON-LD block")
	}
	rest := ld[i+len(marker):]
	j := strings.Index(rest, `</script>`)
	if j < 0 {
		t.Fatal("landing page JSON-LD block is not terminated")
	}
	if strings.Contains(rest[:j], "zzzhidden") {
		t.Error("hidden project leaked into the JSON-LD structured data")
	}
}

func TestAdminProjectFormTemplatesRenderCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	cases := []struct {
		path  string
		forms int
	}{
		{path: "/admin/projects", forms: 1 + len(store.Current().Projects)},
		{path: "/admin/projects/new", forms: 2},
		{path: "/admin/projects/1", forms: 2},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			token := findCookie(t, rec.Result().Cookies(), csrfCookieName).Value
			want := `name="csrf_token" value="` + token + `"`
			if got := strings.Count(rec.Body.String(), want); got != tc.forms {
				t.Errorf("%s renders %d csrf-token hidden fields, want %d", tc.path, got, tc.forms)
			}
		})
	}
}

func TestAdminProjectCreateAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin/projects/new", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	token := csrfTokenForAction(t, getRec.Body.String(), "/admin/projects")

	form := url.Values{
		"csrf_token":  {token},
		"name":        {"csrfproject"},
		"url":         {"https://csrf.example.com"},
		"description": {"created with a rendered token"},
		"icon":        {"globe"},
		"url_label":   {"csrf.example.com"},
		"sort":        {"99"},
		"visible":     {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/projects", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, p := range store.Current().Projects {
		if p.Name == "csrfproject" {
			found = true
		}
	}
	if !found {
		t.Error("created project is missing from the content snapshot")
	}
}

func TestAdminProjectPostRejectsMissingCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	form := url.Values{"name": {"no-csrf"}, "description": {"x"}, "icon": {"globe"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/projects", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a project POST without a CSRF token was accepted", rec.Code)
	}
}
