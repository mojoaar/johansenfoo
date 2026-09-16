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

func TestCSRFRejectsMismatchedToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"csrf_token": {"wrong"}, "name": {"x"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "right"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a mismatched token was accepted", rec.Code)
	}
}

func TestMethodOverrideCannotBypassCSRF(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"_method": {"delete"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a method-overridden state change bypassed CSRF", rec.Code)
	}
}

func TestMethodOverrideCannotSmuggleExemptVerb(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"_method": {"get"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; _method=get smuggled an exempt verb past CSRF", rec.Code)
	}
}

func TestCSRFRejectsMCPPathLookalike(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := csrfMiddleware(inner)

	req := httptest.NewRequest(http.MethodPost, "/mcpxyz", strings.NewReader("x=1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a /mcp lookalike path was exempted", rec.Code)
	}
}

func TestCSRFSkipsExactMCPPath(t *testing.T) {
	for _, path := range []string{"/mcp", "/mcp/"} {
		t.Run(path, func(t *testing.T) {
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			h := csrfMiddleware(inner)

			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("x=1"))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Fatalf("%s was rejected by CSRF", path)
			}
		})
	}
}

func TestCSRFIgnoresQueryStringToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := csrfMiddleware(inner)

	req := httptest.NewRequest(http.MethodPost, "/admin/profile?csrf_token=tok", strings.NewReader("name=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; a query-string token was accepted", rec.Code)
	}
}

func TestCSRFAcceptsMatchingToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := csrfMiddleware(inner)

	form := url.Values{"csrf_token": {"tok"}, "name": {"x"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/profile", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "tok"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; a matching token was rejected", rec.Code)
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

func TestMethodOverrideIgnoresNonMutationVerbs(t *testing.T) {
	for _, verb := range []string{"get", "head", "options"} {
		t.Run(verb, func(t *testing.T) {
			var seen string
			h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = r.Method
			}))
			form := url.Values{"_method": {verb}}
			req := httptest.NewRequest(http.MethodPost, "/admin/projects/3", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			h.ServeHTTP(httptest.NewRecorder(), req)

			if seen != http.MethodPost {
				t.Errorf("_method=%s: method = %q, want %q", verb, seen, http.MethodPost)
			}
		})
	}
}

func TestMethodOverrideRewritesMutationVerbs(t *testing.T) {
	for _, verb := range []string{"put", "patch", "delete"} {
		t.Run(verb, func(t *testing.T) {
			var seen string
			h := methodOverride(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = r.Method
			}))
			form := url.Values{"_method": {verb}}
			req := httptest.NewRequest(http.MethodPost, "/admin/projects/3", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			h.ServeHTTP(httptest.NewRecorder(), req)

			if seen != strings.ToUpper(verb) {
				t.Errorf("_method=%s: method = %q, want %q", verb, seen, strings.ToUpper(verb))
			}
		})
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
