package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func passwordConfigured(t *testing.T, d *sql.DB) bool {
	t.Helper()
	_, err := db.NewSettingsRepo(d).Get(passwordHashKey)
	return err == nil
}

func TestSetupCreatesPasswordAndSession(t *testing.T) {
	d := newTestDB(t)
	store, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	h := newAuthTestHandler(t, d, store)

	if passwordConfigured(t, d) {
		t.Fatal("precondition failed: password already configured")
	}

	form := url.Values{"password": {"correct horse battery staple"}}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if !passwordConfigured(t, d) {
		t.Fatal("password was not stored")
	}
	hash, err := db.NewSettingsRepo(d).Get(passwordHashKey)
	if err != nil {
		t.Fatalf("Get hash: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("correct horse battery staple")); err != nil {
		t.Errorf("stored hash does not verify: %v", err)
	}
	if len(rec.Result().Cookies()) == 0 {
		t.Error("no session cookie was set")
	}
}

func TestSetupRejectsShortPassword(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	form := url.Values{"password": {"short"}}
	req := httptest.NewRequest(http.MethodPost, "/setup", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if passwordConfigured(t, d) {
		t.Error("a short password was accepted")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
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

	form := url.Values{"password": {"wrong-password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName && c.Value != "" {
			t.Error("a session cookie was issued for a failed login")
		}
	}
}

func TestLoginSucceedsAndSeedsSession(t *testing.T) {
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

	form := url.Values{"password": {"right-password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	var sid string
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookieName {
			sid = c.Value
		}
	}
	if sid == "" {
		t.Fatal("no session cookie issued")
	}
	if _, err := db.NewSessionRepo(d).Get(sid); err != nil {
		t.Errorf("session %q not in the database: %v", sid, err)
	}
}

func TestAdminRequiresSession(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Errorf("Location = %q, want %q", loc, "/login")
	}
}

func TestLogoutClearsSession(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	if err := db.NewSessionRepo(d).Create("sess-1", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "sess-1"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if _, err := db.NewSessionRepo(d).Get("sess-1"); err == nil {
		t.Error("session row survived logout")
	}
}

func newAuthTestHandler(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	cfg := &config.Config{Port: 8080, DBPath: "test.db", BaseURL: "https://johansen.foo"}
	return New(Deps{DB: d, Cfg: cfg, Content: store, Version: "test", Started: time.Now()})
}
