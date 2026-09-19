package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func seedViews(t *testing.T, d *db.PageViewRepo, n int) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		path := "/"
		if i%2 == 0 {
			path = "/posts"
		}
		if err := d.Record(&db.PageView{Path: path, Referrer: "https://ref.example", IPHash: "h", CreatedAt: now.Add(-time.Duration(i) * time.Minute)}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
}

func TestAPIAdminStatsVisitorsRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/stats/visitors", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminStatsVisitors(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	seedViews(t, db.NewPageViewRepo(d), 4)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/stats/visitors?period=7d", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeBody[struct {
		Period       string         `json:"period"`
		Views        int            `json:"views"`
		DailyUniques int            `json:"daily_uniques"`
		TopPaths     []db.PathCount `json:"top_paths"`
		Recent       []db.PageView  `json:"recent"`
	}](t, rec)
	if got.Period != "7d" || got.Views != 4 {
		t.Fatalf("stats = %+v", got)
	}
	if len(got.TopPaths) == 0 || got.DailyUniques != 1 {
		t.Errorf("top paths/dailies = %+v / %d", got.TopPaths, got.DailyUniques)
	}
}

func TestAPIAdminStatsClear(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	seedViews(t, db.NewPageViewRepo(d), 3)
	h := loggedInHandler(t, d, store)

	rec := apiDo(t, h, http.MethodDelete, "/api/v1/admin/stats/visitors", "", withSession)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if n := viewCount(t, db.NewPageViewRepo(d)); n != 0 {
		t.Errorf("views = %d after clear, want 0", n)
	}
}

func TestAdminDashboardShowsVisitorStats(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	seedViews(t, db.NewPageViewRepo(d), 2)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Views today") {
		t.Error("dashboard is missing the visitor cards")
	}
}

func TestAdminDashboardShowsDisabledState(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	if err := db.NewSettingsRepo(d).Set("stats_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	rec := adminRequest(t, d, store, http.MethodGet, "/admin", nil)
	if !strings.Contains(rec.Body.String(), "disabled") {
		t.Error("dashboard does not show the disabled state")
	}
}

func TestAdminDashboardShowsRecentHits(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	seedViews(t, db.NewPageViewRepo(d), 1)
	rec := adminRequest(t, d, store, http.MethodGet, "/admin", nil)
	if !strings.Contains(rec.Body.String(), "Recent hits") {
		t.Error("dashboard is missing the recent hits card")
	}
}

func TestAdminStatsSettingsToggle(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	rec := adminRequest(t, d, store, http.MethodPost, "/admin/stats", url.Values{
		"stats_retention_days": {"30"},
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want 303; body=%s", rec.Code, rec.Body.String())
	}
	settings := db.NewSettingsRepo(d)
	if v, _ := settings.Get("stats_enabled"); v != "false" {
		t.Errorf("stats_enabled = %q, want false (unchecked box)", v)
	}
	if v, _ := settings.Get("stats_retention_days"); v != "30" {
		t.Errorf("retention = %q, want 30", v)
	}
}

func TestAPIAdminStatsSettingsPut(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodPut, "/api/v1/admin/settings/stats", `{"enabled":false,"retention_days":14}`, withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	settings := db.NewSettingsRepo(d)
	if v, _ := settings.Get("stats_enabled"); v != "false" {
		t.Errorf("stats_enabled = %q", v)
	}
	if v, _ := settings.Get("stats_retention_days"); v != "14" {
		t.Errorf("retention = %q", v)
	}
}

func TestAPIAdminStatsSystemRequiresAuth(t *testing.T) {
	h := newTestHandler(t)
	if rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/stats/system", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAPIAdminStatsSystem(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := loggedInHandler(t, d, store)
	rec := apiDo(t, h, http.MethodGet, "/api/v1/admin/stats/system", "", withSession)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeBody[struct {
		Goroutines    int   `json:"goroutines"`
		UptimeSeconds int64 `json:"uptime_seconds"`
	}](t, rec)
	if got.Goroutines <= 0 {
		t.Errorf("goroutines = %d, want > 0", got.Goroutines)
	}
}
