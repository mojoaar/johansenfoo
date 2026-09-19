package web

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
)

func newTestHandlerWith(t *testing.T, d *sql.DB, store *ContentStore) http.Handler {
	t.Helper()
	return New(Deps{
		DB:      d,
		Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
		Content: store,
		Version: "test",
		Started: time.Now(),
	})
}

func apiDo(t *testing.T, h http.Handler, method, path, body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func withSession(req *http.Request) {
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: "test-session"})
}

func withBearer(key string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode body: %v; body=%q", err, rec.Body.String())
	}
	return v
}

func decodeJSONBytes[T any](t *testing.T, body []byte) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(body, &v); err != nil {
		t.Fatalf("decode bytes: %v; body=%q", err, string(body))
	}
	return v
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
