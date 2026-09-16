package web

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func clearLoginLimiter() {
	loginLimiter.mu.Lock()
	loginLimiter.entries = make(map[string]*rateEntry)
	loginLimiter.mu.Unlock()
}

func resetLoginLimiter(t *testing.T) {
	t.Helper()
	clearLoginLimiter()
	t.Cleanup(clearLoginLimiter)
}

func authTestPost(path string, form url.Values, remoteAddr string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	return req
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("cookie %q was not set; got %v", name, cookies)
	return nil
}

func TestExpiredSessionIsRejected(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	if err := db.NewSessionRepo(d).Create("expired-session", time.Now().UTC().Add(-time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "expired-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

func TestValidSessionReachesAdmin(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	id := randomToken()
	if err := db.NewSessionRepo(d).Create(id, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: id})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<h1>Dashboard</h1>") {
		t.Errorf("body does not contain the dashboard heading: %s", body)
	}
	if !strings.Contains(body, `action="/logout"`) {
		t.Errorf("body does not contain the sign-out form: %s", body)
	}
}

func TestSetupRefusesWhenAlreadyConfigured(t *testing.T) {
	d := newTestDB(t)
	original, err := bcrypt.GenerateFromPassword([]byte("existing-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(original)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"password": {"replacement-password"}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authTestPost("/setup", form, ""))

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
	after, err := db.NewSettingsRepo(d).Get(passwordHashKey)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if after != string(original) {
		t.Error("stored password hash was overwritten after setup was already complete")
	}
}

func TestSetupRejectsPasswordConfirmMismatch(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{
		"password":         {"correct horse battery staple"},
		"password_confirm": {"correct horse battery stapler"},
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authTestPost("/setup", form, ""))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if passwordConfigured(t, d) {
		t.Error("a mismatched password confirmation still stored a password")
	}
}

func TestLoginRateLimitBlocksExtraAttempts(t *testing.T) {
	resetLoginLimiter(t)
	d := newTestDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("right-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set(passwordHashKey, string(hash)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	const remoteAddr = "198.51.100.7:54321"
	const allowedAttempts = 10
	if got := clientIP(&http.Request{RemoteAddr: remoteAddr}); got != "198.51.100.7" {
		t.Fatalf("clientIP = %q, want %q", got, "198.51.100.7")
	}

	for i := 0; i < allowedAttempts; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, authTestPost("/login", url.Values{"password": {"wrong-password"}}, remoteAddr))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want %d; body=%s", i+1, rec.Code, http.StatusUnauthorized, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authTestPost("/login", url.Values{"password": {"right-password"}}, remoteAddr))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("attempt %d: status = %d, want %d; body=%s", allowedAttempts+1, rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
}

func TestSecureRequestTLSBranch(t *testing.T) {
	if !secureRequest(&http.Request{TLS: &tls.ConnectionState{}}) {
		t.Error("secureRequest = false for a request carrying TLS state, want true")
	}
	if !secureRequest(&http.Request{Header: http.Header{"X-Forwarded-Proto": {"https"}}}) {
		t.Error("secureRequest = false for an X-Forwarded-Proto: https request, want true")
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	cases := []struct {
		name   string
		secure bool
	}{
		{name: "plain-http", secure: false},
		{name: "tls-terminated", secure: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, _ := NewContentStore(d)
			h := newAuthTestHandler(t, d, store)

			form := url.Values{
				"password":         {"correct horse battery staple"},
				"password_confirm": {"correct horse battery staple"},
			}
			req := authTestPost("/setup", form, "")
			if tc.secure {
				req.Header.Set("X-Forwarded-Proto", "https")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			c := findCookie(t, rec.Result().Cookies(), sessionCookieName)
			if c.Path != "/" {
				t.Errorf("Path = %q, want %q", c.Path, "/")
			}
			if !c.HttpOnly {
				t.Error("HttpOnly = false, want true")
			}
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("SameSite = %v, want %v", c.SameSite, http.SameSiteLaxMode)
			}
			if c.Secure != tc.secure {
				t.Errorf("Secure = %v, want %v", c.Secure, tc.secure)
			}
		})
	}
}

func TestCSRFCookieAttributes(t *testing.T) {
	cases := []struct {
		name   string
		secure bool
	}{
		{name: "plain-http", secure: false},
		{name: "tls-terminated", secure: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, _ := NewContentStore(d)
			h := newAuthTestHandler(t, d, store)

			req := httptest.NewRequest(http.MethodGet, "/setup", nil)
			if tc.secure {
				req.Header.Set("X-Forwarded-Proto", "https")
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			c := findCookie(t, rec.Result().Cookies(), csrfCookieName)
			if c.Path != "/" {
				t.Errorf("Path = %q, want %q", c.Path, "/")
			}
			if !c.HttpOnly {
				t.Error("HttpOnly = false, want true")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Errorf("SameSite = %v, want %v", c.SameSite, http.SameSiteStrictMode)
			}
			if c.Secure != tc.secure {
				t.Errorf("Secure = %v, want %v", c.Secure, tc.secure)
			}
		})
	}
}
