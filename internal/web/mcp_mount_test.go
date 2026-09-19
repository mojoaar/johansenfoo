package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestMCPMountedAndRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (mounted and CSRF-exempt)", rec.Code)
	}
}

func TestMCPAuthenticatedInitializeThroughRouter(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	if err := db.NewSettingsRepo(d).Set("api_key", "k"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	h := newTestHandlerWith(t, d, store)

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"c","version":"0"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer k")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "serverInfo") {
		t.Errorf("authenticated initialize body missing serverInfo: %s", rec.Body.String())
	}
}
