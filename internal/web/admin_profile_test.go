package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminRequest(t *testing.T, d *sql.DB, store *ContentStore, method, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	h := loggedInHandler(t, d, store)
	var body *strings.Reader
	if form == nil {
		body = strings.NewReader("")
	} else {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func TestAdminProfileGetRendersCurrentValues(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/profile", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Morten Johansen", "mojoaar", "Denmark", `name="hero_bio"`, `name="bio"`} {
		if !strings.Contains(body, want) {
			t.Errorf("profile form missing %q", want)
		}
	}
}

func TestAdminProfilePostUpdatesAndReloads(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	form := url.Values{
		"name": {"Changed Name"}, "handle": {"mojoaar"}, "location": {"Denmark"},
		"dob": {"1980-08-13"}, "tagline": {"// changed"}, "hero_bio": {"hero"},
		"bio": {"bio"}, "about_para_1": {"one"}, "about_para_2": {"two"}, "avatar": {"/static/avatar.png"},
	}
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/profile", form)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	if got := store.Current().Profile.Name; got != "Changed Name" {
		t.Errorf("snapshot Profile.Name = %q, want %q", got, "Changed Name")
	}
	p, err := db.NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "Changed Name" {
		t.Errorf("stored Name = %q", p.Name)
	}
	if p.Bio != "bio" {
		t.Errorf("stored Bio = %q, want %q", p.Bio, "bio")
	}
}

func TestAdminSocialCreateUpdateDelete(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/social", url.Values{
		"platform": {"example"}, "url": {"https://example.com"}, "label": {"Example"}, "sort": {"99"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("create status = %d, want 303", rec.Code)
	}
	links, err := db.NewProfileRepo(d).SocialLinks()
	if err != nil {
		t.Fatalf("SocialLinks: %v", err)
	}
	var id int64
	for _, l := range links {
		if l.Platform == "example" {
			id = l.ID
		}
	}
	if id == 0 {
		t.Fatal("social link was not created")
	}
	if len(store.Current().Social) != 5 {
		t.Errorf("snapshot has %d social links, want 5", len(store.Current().Social))
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/social/"+itoa(id), url.Values{
		"platform": {"example"}, "url": {"https://example.com"}, "label": {"Renamed"}, "sort": {"99"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("update status = %d, want 303", rec.Code)
	}
	l, _ := db.NewProfileRepo(d).SocialLink(id)
	if l.Label != "Renamed" {
		t.Errorf("Label = %q, want %q", l.Label, "Renamed")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/social/"+itoa(id)+"/delete", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("delete status = %d, want 303", rec.Code)
	}
	if len(store.Current().Social) != 4 {
		t.Errorf("snapshot has %d social links after delete, want 4", len(store.Current().Social))
	}
}

func socialFromSnapshot(t *testing.T, store *ContentStore, id int64) db.SocialLink {
	t.Helper()
	for _, l := range store.Current().Social {
		if l.ID == id {
			return l
		}
	}
	t.Fatalf("social link %d is missing from the content snapshot", id)
	return db.SocialLink{}
}

func TestAdminSocialUpdateRefreshesContentSnapshot(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	id, err := db.NewProfileRepo(d).CreateSocialLink(&db.SocialLink{
		Platform: "snapshot", URL: "https://old.example.com", Label: "Old label", Sort: 99, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreateSocialLink: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := socialFromSnapshot(t, store, id); got.URL != "https://old.example.com" {
		t.Fatalf("precondition: snapshot URL = %q, want %q", got.URL, "https://old.example.com")
	}

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/social/"+itoa(id), url.Values{
		"platform": {"snapshot"}, "url": {"https://new.example.com"}, "label": {"New label"}, "sort": {"99"}, "visible": {"1"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}

	got := socialFromSnapshot(t, store, id)
	if got.URL != "https://new.example.com" {
		t.Errorf("snapshot URL = %q, want %q", got.URL, "https://new.example.com")
	}
	if got.Label != "New label" {
		t.Errorf("snapshot Label = %q, want %q", got.Label, "New label")
	}
}

func csrfTokenForAction(t *testing.T, body, action string) string {
	t.Helper()
	const actionMarker = `action="`
	i := strings.Index(body, actionMarker+action+`"`)
	if i < 0 {
		t.Fatalf("form %q is missing from the rendered page", action)
	}
	rest := body[i:]
	const tokenMarker = `name="csrf_token" value="`
	j := strings.Index(rest, tokenMarker)
	if j < 0 {
		t.Fatalf("form %q has no csrf_token hidden field", action)
	}
	rest = rest[j+len(tokenMarker):]
	k := strings.IndexByte(rest, '"')
	if k < 0 {
		t.Fatalf("form %q has a malformed csrf_token hidden field", action)
	}
	return rest[:k]
}

func TestAdminFormTemplatesRenderCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	cases := []struct {
		path  string
		forms int
	}{
		{path: "/admin/profile", forms: 2},
		{path: "/admin/social", forms: 2 + 2*len(store.Current().Social)},
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

func TestAdminProfilePostAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin/profile", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	token := csrfTokenForAction(t, getRec.Body.String(), "/admin/profile")

	form := url.Values{
		"csrf_token": {token},
		"name":       {"Via rendered CSRF"},
		"handle":     {"mojoaar"},
		"location":   {"Denmark"},
		"tagline":    {"// csrf"},
		"hero_bio":   {"hero"},
		"bio":        {"bio"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	if got := store.Current().Profile.Name; got != "Via rendered CSRF" {
		t.Errorf("snapshot Profile.Name = %q, want %q", got, "Via rendered CSRF")
	}
}

func TestAdminSocialCreateAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin/social", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, get)
	token := csrfTokenForAction(t, getRec.Body.String(), "/admin/social")

	form := url.Values{
		"csrf_token": {token},
		"platform":   {"csrfexample"},
		"url":        {"https://csrf.example.com"},
		"label":      {"CSRF Example"},
		"sort":       {"99"},
		"visible":    {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/social", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	found := false
	for _, l := range store.Current().Social {
		if l.Platform == "csrfexample" {
			found = true
		}
	}
	if !found {
		t.Error("created social link is missing from the content snapshot")
	}
}

func TestAdminSocialPostRejectsMissingCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	form := url.Values{"platform": {"no-csrf"}, "url": {"https://no.example.com"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/social", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a social POST without a CSRF token was accepted", rec.Code)
	}
}
