package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestMetricsEndpoint(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/metrics", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "go_goroutines") {
		t.Error("metrics body is missing go_goroutines")
	}
}

func TestMetricsNotRecorded(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	apiDo(t, h, http.MethodGet, "/metrics", "", nil)
	if n := viewCount(t, db.NewPageViewRepo(d)); n != 0 {
		t.Errorf("views = %d, want 0 (metrics must not be recorded)", n)
	}
}

func TestCollectRuntime(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	deps := Deps{DB: d, Cfg: &config.Config{DBPath: "/tmp/jf-test.db"}, Content: store, Version: "test", Started: time.Now().Add(-time.Minute)}
	got := collectRuntime(deps)
	if got.Goroutines <= 0 {
		t.Errorf("goroutines = %d, want > 0", got.Goroutines)
	}
	if got.UptimeSeconds < 0 {
		t.Errorf("uptime = %d, want >= 0", got.UptimeSeconds)
	}
}
