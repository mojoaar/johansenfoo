package mcp

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := db.Migrate(d); err != nil {
		t.Fatalf("db.Migrate: %v", err)
	}
	return d
}

func testDeps(t *testing.T, key string) Deps {
	t.Helper()
	return Deps{
		DB:     testDB(t),
		Reload: func() error { return nil },
		APIKey: func() (string, error) { return key, nil },
	}
}

func initializeRequest(t *testing.T, h http.Handler, mutate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestMCPRequiresBearer(t *testing.T) {
	h := Handler(testDeps(t, "secret"))
	if rec := initializeRequest(t, h, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestMCPRejectsWrongKey(t *testing.T) {
	h := Handler(testDeps(t, "secret"))
	rec := initializeRequest(t, h, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer wrong")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong key status = %d, want 401", rec.Code)
	}
	rec = initializeRequest(t, h, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer secret")
	})
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("right key rejected: %d", rec.Code)
	}
}

func TestMCPInitialize(t *testing.T) {
	h := Handler(testDeps(t, "secret"))
	rec := initializeRequest(t, h, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer secret")
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "serverInfo") {
		t.Fatalf("initialize response missing serverInfo: %s", rec.Body.String())
	}
}

func TestMCPRateLimited(t *testing.T) {
	oldMax, oldWindow := rateLimitMax, rateLimitWindow
	rateLimitMax, rateLimitWindow = 2, time.Minute
	t.Cleanup(func() { rateLimitMax, rateLimitWindow = oldMax, oldWindow })

	h := Handler(testDeps(t, "secret"))
	auth := func(r *http.Request) { r.Header.Set("Authorization", "Bearer secret") }
	for i := 0; i < 2; i++ {
		if rec := initializeRequest(t, h, auth); rec.Code == http.StatusTooManyRequests {
			t.Fatalf("request %d was rate limited too early", i+1)
		}
	}
	if rec := initializeRequest(t, h, auth); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want 429", rec.Code)
	}
}

func TestMCPNoKeyConfigured(t *testing.T) {
	h := Handler(testDeps(t, ""))
	rec := initializeRequest(t, h, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer anything")
	})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when no key is configured", rec.Code)
	}
}
