package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func loggedInHandler(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	sessions := db.NewSessionRepo(d)
	if _, err := sessions.Get("test-session"); err != nil {
		if err := sessions.Create("test-session", time.Now().UTC().Add(time.Hour)); err != nil {
			t.Fatalf("Create session: %v", err)
		}
	}
	return newAuthTestHandler(t, d, store)
}

func TestDashboardRendersWithVendoredHTMX(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{`src="/static/htmx.min.js"`, "Dashboard", `href="/admin/projects"`, `action="/logout"`} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard body missing %q", want)
		}
	}
	if strings.Contains(body, "unpkg.com") || strings.Contains(body, "cdn.") {
		t.Error("dashboard loads a script from a third-party origin")
	}
}

func TestDashboardShowsEntityCounts(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `<span class="stat-value">9</span>`) {
		t.Errorf("dashboard does not report the project count")
	}
	if !strings.Contains(body, `<span class="stat-value">30</span>`) {
		t.Errorf("dashboard does not report the skill count")
	}
}

func TestAdminPagesAreNoIndex(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Body.String(); !strings.Contains(got, `content="noindex"`) {
		t.Error("admin page is missing robots noindex")
	}
}

func TestDashboardLogoutFormCarriesMatchingCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	get := httptest.NewRequest(http.MethodGet, "/admin", nil)
	get.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, get)

	var csrf string
	for _, c := range rec.Result().Cookies() {
		if c.Name == csrfCookieName {
			csrf = c.Value
		}
	}
	if csrf == "" {
		t.Fatal("no CSRF cookie issued with the dashboard")
	}
	if !strings.Contains(rec.Body.String(), `name="csrf_token" value="`+csrf+`"`) {
		t.Fatal("logout form does not carry the CSRF token")
	}

	form := url.Values{"csrf_token": {csrf}}
	post := httptest.NewRequest(http.MethodPost, "/logout", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	post.AddCookie(&http.Cookie{Name: csrfCookieName, Value: csrf})
	postRec := httptest.NewRecorder()
	h.ServeHTTP(postRec, post)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("logout status = %d, want %d; body=%s", postRec.Code, http.StatusSeeOther, postRec.Body.String())
	}
	if _, err := db.NewSessionRepo(d).Get("test-session"); err == nil {
		t.Error("session survived a CSRF-protected logout")
	}
}
