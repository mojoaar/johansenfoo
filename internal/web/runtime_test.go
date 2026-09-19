package web

import (
	"net/http"
	"net/http/httptest"
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

func TestAdminRuntimePartial(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/runtime", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Goroutines") {
		t.Error("partial is missing the goroutines card")
	}
	if strings.Contains(body, "<html") {
		t.Error("partial returned a full page, not a fragment")
	}
}

func TestAdminRuntimeRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	rec := apiDo(t, h, http.MethodGet, "/admin/runtime", "", nil)
	if rec.Code == http.StatusOK {
		t.Fatalf("status = %d, want a redirect/unauthorized", rec.Code)
	}
}

func TestDashboardPollsRuntime(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin", nil)
	if !strings.Contains(rec.Body.String(), `hx-get="/admin/runtime"`) {
		t.Error("dashboard does not poll the runtime partial")
	}
}

func TestRuntimePartialHasHeadingAndDiskTotal(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin/runtime", nil)
	if !strings.Contains(rec.Body.String(), "Runtime") {
		t.Error("partial is missing the Runtime heading")
	}

	w := httptest.NewRecorder()
	renderFragment(w, "admin_runtime", page{Runtime: RuntimeStats{Available: true, DiskUsed: 1, DiskTotal: 2}})
	if !strings.Contains(w.Body.String(), "disk total") {
		t.Error("partial is missing disk total when container stats are available")
	}
}

func TestNewWithNilCfgDoesNotPanic(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("New panicked with nil Cfg: %v", r)
		}
	}()
	_ = New(Deps{DB: d, Content: store, Version: "test", Started: time.Now()})
}
