package web

import (
	"database/sql"
	"net/http"
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
