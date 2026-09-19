package web

import (
	"net/http"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func viewCount(t *testing.T, d *db.PageViewRepo) int {
	t.Helper()
	n, err := d.CountSince("2000-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("CountSince: %v", err)
	}
	return n
}

func TestHashIPStableAndSaltDependent(t *testing.T) {
	a := hashIP("salt-a", "1.2.3.4")
	if a != hashIP("salt-a", "1.2.3.4") {
		t.Fatal("hash is not stable for the same salt and ip")
	}
	if a == hashIP("salt-b", "1.2.3.4") {
		t.Fatal("hash does not depend on the salt")
	}
	if a == hashIP("salt-a", "5.6.7.8") {
		t.Fatal("hash does not depend on the ip")
	}
}

func TestRecorderRotatesSaltByDay(t *testing.T) {
	d := newTestDB(t)
	rec := newViewRecorder(d)
	first, err := rec.saltFor("2026-09-19")
	if err != nil {
		t.Fatalf("saltFor: %v", err)
	}
	same, err := rec.saltFor("2026-09-19")
	if err != nil {
		t.Fatalf("saltFor: %v", err)
	}
	if first != same {
		t.Fatal("salt is not stable within a day")
	}
	second, err := rec.saltFor("2026-09-20")
	if err != nil {
		t.Fatalf("saltFor: %v", err)
	}
	if first == second {
		t.Fatal("salt did not rotate when the day changed")
	}
}

func TestPageViewRecordsPublicHTML(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)

	rec := apiDo(t, h, http.MethodGet, "/", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	repo := db.NewPageViewRepo(d)
	if n := viewCount(t, repo); n != 1 {
		t.Fatalf("views = %d, want 1", n)
	}
	recent, err := repo.Recent(1)
	if err != nil || len(recent) != 1 {
		t.Fatalf("Recent = %+v, err=%v", recent, err)
	}
	if recent[0].IPHash == "" || recent[0].IPHash == "192.0.2.1" {
		t.Errorf("ip_hash = %q, want a salted hash (never the raw IP)", recent[0].IPHash)
	}
}

func TestPageViewIgnoresJSONErrorsAndPosts(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	repo := db.NewPageViewRepo(d)

	apiDo(t, h, http.MethodGet, "/me", "", nil)
	if n := viewCount(t, repo); n != 0 {
		t.Errorf("JSON response was recorded (%d rows)", n)
	}
	apiDo(t, h, http.MethodGet, "/nope", "", nil)
	if n := viewCount(t, repo); n != 0 {
		t.Errorf("404 was recorded (%d rows)", n)
	}
	apiDo(t, h, http.MethodPost, "/login", "password=x", nil)
	if n := viewCount(t, repo); n != 0 {
		t.Errorf("POST was recorded (%d rows)", n)
	}
	apiDo(t, h, http.MethodGet, "/admin", "", nil)
	if n := viewCount(t, repo); n != 0 {
		t.Errorf("admin request was recorded (%d rows)", n)
	}
}

func TestPageViewDisabled(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newTestHandlerWith(t, d, store)
	if err := db.NewSettingsRepo(d).Set("stats_enabled", "false"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	apiDo(t, h, http.MethodGet, "/", "", nil)
	if n := viewCount(t, db.NewPageViewRepo(d)); n != 0 {
		t.Errorf("views = %d, want 0 while stats are disabled", n)
	}
}

func TestPruneViewsUsesRetentionSetting(t *testing.T) {
	d := newTestDB(t)
	repo := db.NewPageViewRepo(d)
	old := time.Now().UTC().AddDate(0, 0, -10)
	if err := repo.Record(&db.PageView{Path: "/old", IPHash: "x", CreatedAt: old}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := repo.Record(&db.PageView{Path: "/new", IPHash: "y"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := db.NewSettingsRepo(d).Set("stats_retention_days", "5"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	n, err := pruneViews(d)
	if err != nil {
		t.Fatalf("pruneViews: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned = %d, want 1", n)
	}
	if remaining := viewCount(t, repo); remaining != 1 {
		t.Errorf("remaining = %d, want 1", remaining)
	}
}
