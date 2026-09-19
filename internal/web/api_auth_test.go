package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestAPIAdminRequiresCredentials(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminAcceptsBearerKey(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	if err := db.NewSettingsRepo(d).Set(apiKeySettingKey, "secret-key"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	h := newTestHandlerWith(t, d, store)

	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("wrong")); rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status = %d, want 401", rec.Code)
	}
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("secret-key")); rec.Code != http.StatusOK {
		t.Fatalf("right key status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAPIAdminRejectsBearerWhenNoKeyConfigured(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withBearer("anything"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminAcceptsSessionCookie(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/profile", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestCSRFExemptsAPIPaths(t *testing.T) {
	h := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/projects", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (API paths must be CSRF-exempt)", rec.Code)
	}
}
