package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	return New(Deps{
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Version: "test",
		Started: time.Now(),
	})
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "ok\n" {
		t.Errorf("body = %q, want %q", got, "ok\n")
	}
}
