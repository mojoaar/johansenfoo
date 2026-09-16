package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAdminExperienceCreateAndDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/experience", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Jydske Dragonregiment") {
		t.Error("experience list is missing a seeded entry")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/experience", url.Values{
		"years": {"1990-1991"}, "role": {"Tester"}, "company": {"ACME"},
		"icon": {"briefcase"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	if len(store.Current().Experience) != 9 {
		t.Fatalf("snapshot experience = %d, want 9", len(store.Current().Experience))
	}

	var id int64
	for _, e := range store.Current().Experience {
		if e.Company == "ACME" {
			id = e.ID
		}
	}
	rec = adminRequest(t, d, store, http.MethodPost, "/admin/experience/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Experience) != 8 {
		t.Errorf("snapshot experience = %d, want 8", len(store.Current().Experience))
	}
}

func TestAdminExperienceEditPrefillsValues(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/experience/1", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `value="2022 – present"`) {
		t.Error("experience edit form does not prefill the years")
	}
}

func experienceFromSnapshot(t *testing.T, store *ContentStore, id int64) db.Experience {
	t.Helper()
	for _, e := range store.Current().Experience {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("experience %d is missing from the content snapshot", id)
	return db.Experience{}
}

func TestAdminExperienceUpdateRefreshesContentSnapshot(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	id, err := db.NewContentRepo(d).CreateExperience(&db.Experience{
		Years: "old years", Role: "old role", Company: "snapshotco", Icon: "briefcase", Sort: 99, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := experienceFromSnapshot(t, store, id); got.Role != "old role" {
		t.Fatalf("precondition: snapshot Role = %q, want %q", got.Role, "old role")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/experience/"+itoa(id), url.Values{
		"years": {"new years"}, "role": {"new role"}, "company": {"snapshotco"},
		"icon": {"briefcase"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	got := experienceFromSnapshot(t, store, id)
	if got.Role != "new role" {
		t.Errorf("snapshot Role = %q, want %q", got.Role, "new role")
	}
	if got.Years != "new years" {
		t.Errorf("snapshot Years = %q, want %q", got.Years, "new years")
	}
}

func TestAdminExperienceHiddenRowsStayOutOfPublicSite(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	id, err := db.NewContentRepo(d).CreateExperience(&db.Experience{
		Years: "1990-1991", Role: "zzzhiddenrole", Company: "zzzhiddenco", Icon: "briefcase", Sort: 50, Visible: false,
	})
	if err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	found := false
	for _, e := range store.Current().Experience {
		if e.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatal("hidden experience is missing from the admin content snapshot")
	}

	adminGet := httptest.NewRequest(http.MethodGet, "/admin/experience", nil)
	adminGet.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	adminRec := httptest.NewRecorder()
	h.ServeHTTP(adminRec, adminGet)
	if !strings.Contains(adminRec.Body.String(), "zzzhiddenco") {
		t.Error("admin experience list does not show the hidden row for re-enabling")
	}

	landing := httptest.NewRequest(http.MethodGet, "/", nil)
	landingRec := httptest.NewRecorder()
	h.ServeHTTP(landingRec, landing)
	if strings.Contains(landingRec.Body.String(), "zzzhiddenco") {
		t.Error("hidden experience leaked onto /")
	}
	if strings.Contains(landingRec.Body.String(), "zzzhiddenrole") {
		t.Error("hidden experience role leaked onto /")
	}

	me := httptest.NewRequest(http.MethodGet, "/me", nil)
	meRec := httptest.NewRecorder()
	h.ServeHTTP(meRec, me)
	if strings.Contains(meRec.Body.String(), "zzzhiddenco") {
		t.Error("hidden experience leaked into /me")
	}
}

func TestAdminExperienceVisibilityReachesPublicSite(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	id, err := db.NewContentRepo(d).CreateExperience(&db.Experience{
		Years: "1990-1991", Role: "toggle role", Company: "zzztoggleco", Icon: "briefcase", Sort: 50, Visible: false,
	})
	if err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if experienceFromSnapshot(t, store, id).Visible {
		t.Fatal("precondition: snapshot Visible = true, want false")
	}
	if strings.Contains(publicLandingBody(t, h), "zzztoggleco") {
		t.Fatal("precondition: hidden experience already appears on /")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/experience/"+itoa(id), url.Values{
		"years": {"1990-1991"}, "role": {"toggle role"}, "company": {"zzztoggleco"},
		"icon": {"briefcase"}, "sort": {"50"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("enable status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if !experienceFromSnapshot(t, store, id).Visible {
		t.Error("snapshot Visible = false after enabling, want true")
	}
	if !strings.Contains(publicLandingBody(t, h), "zzztoggleco") {
		t.Error("newly visible experience does not appear on /")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/experience/"+itoa(id), url.Values{
		"years": {"1990-1991"}, "role": {"toggle role"}, "company": {"zzztoggleco"},
		"icon": {"briefcase"}, "sort": {"50"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("disable status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if experienceFromSnapshot(t, store, id).Visible {
		t.Error("snapshot Visible = true after disabling, want false")
	}
	if strings.Contains(publicLandingBody(t, h), "zzztoggleco") {
		t.Error("hidden experience still appears on /")
	}
}

func TestAdminExperienceFormTemplatesRenderCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	cases := []struct {
		path  string
		forms int
	}{
		{path: "/admin/experience", forms: 1 + len(store.Current().Experience)},
		{path: "/admin/experience/new", forms: 2},
		{path: "/admin/experience/1", forms: 2},
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

func TestAdminExperienceCreateAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin/experience/new", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	token := csrfTokenForAction(t, getRec.Body.String(), "/admin/experience")

	form := url.Values{
		"csrf_token": {token},
		"years":      {"2020-2021"},
		"role":       {"csrf tester"},
		"company":    {"csrfco"},
		"icon":       {"briefcase"},
		"sort":       {"99"},
		"visible":    {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/experience", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, e := range store.Current().Experience {
		if e.Company == "csrfco" {
			found = true
		}
	}
	if !found {
		t.Error("created experience entry is missing from the content snapshot")
	}
}

func TestAdminExperiencePostRejectsMissingCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	form := url.Values{"years": {"2020-2021"}, "role": {"no-csrf"}, "company": {"no-csrf"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/experience", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; an experience POST without a CSRF token was accepted", rec.Code)
	}
}
