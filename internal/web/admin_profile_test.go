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
