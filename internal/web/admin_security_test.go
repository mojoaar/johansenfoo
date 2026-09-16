package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func setPassword(t *testing.T, d *sql.DB, pw string) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(hash)); err != nil {
		t.Fatalf("Set: %v", err)
	}
}

func TestSecurityPageHidesTheKeyWhenNoneExists(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/security", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "No API key") {
		t.Error("security page does not show the empty API key state")
	}
}

func TestSecurityPageReportsExistingKeyAsConfigured(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	const storedKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := db.NewSettingsRepo(d).Set(apiKeySettingKey, storedKey); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}

	rec := adminRequest(t, d, store, http.MethodGet, "/admin/security", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, "No API key") {
		t.Error("security page reports no key although one is stored")
	}
	if !strings.Contains(body, "Regenerate key") {
		t.Error("security page does not offer to regenerate the existing key")
	}
	if strings.Contains(body, "Generate key") {
		t.Error("security page offers Generate key although one is stored")
	}
	if !strings.Contains(body, "replaces it immediately") {
		t.Error("security page omits the regenerate warning for a hidden key")
	}
	if strings.Contains(body, storedKey) {
		t.Error("security page revealed the stored key without ?key=1")
	}
}

func TestAPIKeyRegenerateStoresAndReveals(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/apikey", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303", rec.Code)
	}
	key, err := db.NewSettingsRepo(d).Get("api_key")
	if err != nil {
		t.Fatalf("Get api_key: %v", err)
	}
	if len(key) != 64 {
		t.Errorf("api_key length = %d, want 64 hex chars", len(key))
	}

	rec = adminRequest(t, d, store, http.MethodGet, "/admin/security?key=1", nil)
	if !strings.Contains(rec.Body.String(), key) {
		t.Error("revealed key is not shown on the security page")
	}

	rec = adminRequest(t, d, store, http.MethodPost, "/admin/security/apikey", nil)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("second regenerate status = %d, want 303", rec.Code)
	}
	key2, _ := db.NewSettingsRepo(d).Get("api_key")
	if key2 == key {
		t.Error("regenerate produced the same key")
	}
}

func TestPasswordChangeRequiresCurrentPassword(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"wrong-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-long-enough-new-one"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	hash, _ := db.NewSettingsRepo(d).Get(passwordHashKey)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("original-password")) != nil {
		t.Error("password changed despite a wrong current password")
	}
}

func TestPasswordChangeSucceedsAndKeepsSession(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"original-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-long-enough-new-one"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	hash, _ := db.NewSettingsRepo(d).Get(passwordHashKey)
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte("a-long-enough-new-one")) != nil {
		t.Error("new password does not verify")
	}
	if _, err := db.NewSessionRepo(d).Get("test-session"); err != nil {
		t.Errorf("the acting session was destroyed: %v", err)
	}
}

func TestPasswordChangeRejectsMismatch(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)

	rec := adminRequest(t, d, store, http.MethodPost, "/admin/security/password", url.Values{
		"current_password": {"original-password"},
		"new_password":     {"a-long-enough-new-one"},
		"confirm_password": {"a-different-one-entirely"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func securityPageGet(t *testing.T, h http.Handler) (string, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/admin/security", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /admin/security status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	return rec.Body.String(), findCookie(t, rec.Result().Cookies(), csrfCookieName).Value
}

func TestAdminSecurityFormTemplatesRenderCSRFToken(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	body, token := securityPageGet(t, h)
	want := `name="csrf_token" value="` + token + `"`
	if got := strings.Count(body, want); got != 3 {
		t.Errorf("/admin/security renders %d csrf-token hidden fields, want 3", got)
	}
}

func TestAdminSecurityPostAcceptsRenderedCSRFToken(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	body, cookieToken := securityPageGet(t, h)

	cases := []struct {
		path string
		form url.Values
	}{
		{path: "/admin/security/password", form: url.Values{
			"current_password": {"original-password"},
			"new_password":     {"a-long-enough-new-one"},
			"confirm_password": {"a-long-enough-new-one"},
		}},
		{path: "/admin/security/apikey", form: url.Values{}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			token := csrfTokenForAction(t, body, tc.path)
			if token != cookieToken {
				t.Fatalf("form %q token does not match the issued CSRF cookie", tc.path)
			}
			form := url.Values{"csrf_token": {token}}
			for k, v := range tc.form {
				form[k] = v
			}
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: cookieToken})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusSeeOther {
				t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAdminSecurityPostRejectsMissingCSRF(t *testing.T) {
	d := newTestDB(t)
	setPassword(t, d, "original-password")
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)

	cases := []struct {
		path string
		form url.Values
	}{
		{path: "/admin/security/password", form: url.Values{
			"current_password": {"original-password"},
			"new_password":     {"a-long-enough-new-one"},
			"confirm_password": {"a-long-enough-new-one"},
		}},
		{path: "/admin/security/apikey", form: url.Values{}},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403; a security POST without a CSRF token was accepted", rec.Code)
			}
		})
	}
}
