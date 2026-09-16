package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFRejectsPostWithoutToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestCSRFAcceptsMatchingToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"csrf_token": {"tok"}, "name": {"x"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatalf("status = 403; a matching token was rejected")
	}
}

func TestCSRFSkippedForHTMXRequests(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatal("HTMX request was rejected by CSRF")
	}
}

func TestCSRFSkippedForLoginPath(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("password=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code == http.StatusForbidden {
		t.Fatal("/login was rejected by CSRF")
	}
}

func TestMethodOverrideRewritesPostToDelete(t *testing.T) {
	var seen string
	h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method
	}))
	form := url.Values{"_method": {"delete"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/projects/3", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != http.MethodDelete {
		t.Errorf("method = %q, want %q", seen, http.MethodDelete)
	}
}

func TestMethodOverrideLeavesPlainPostAlone(t *testing.T) {
	var seen string
	h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method
	}))
	req := httptest.NewRequest(http.MethodPost, "/admin/projects", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != http.MethodPost {
		t.Errorf("method = %q, want %q", seen, http.MethodPost)
	}
}
