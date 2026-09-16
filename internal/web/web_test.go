package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	d := newTestDB(t)

	content, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}

	return New(Deps{
		DB:      d,
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Content: content,
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

func TestStaticStylesheetServed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
	body := rec.Body.String()
	if strings.Contains(body, ":root {") {
		t.Error("stylesheet still declares its own token block")
	}
	if !strings.Contains(body, `[data-mode="light"]`) {
		t.Error("stylesheet does not use the data-mode axis")
	}
}

func TestStaticAvatarServed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/static/avatar.png", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Body.Len() == 0 {
		t.Error("avatar.png is empty")
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := db.Migrate(d); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return d
}
