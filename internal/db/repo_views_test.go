package db

import (
	"testing"
	"time"
)

func recordView(t *testing.T, r *PageViewRepo, path, ref, hash string, at time.Time) {
	t.Helper()
	if err := r.Record(&PageView{Path: path, Referrer: ref, IPHash: hash, CreatedAt: at}); err != nil {
		t.Fatalf("Record: %v", err)
	}
}

func TestPageViewRecordAndCount(t *testing.T) {
	d := seeded(t)
	r := NewPageViewRepo(d)
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	recordView(t, r, "/", "", "aaa", now)
	recordView(t, r, "/posts", "", "bbb", now.Add(-time.Hour))
	recordView(t, r, "/old", "", "ccc", now.AddDate(0, 0, -10))

	n, err := r.CountSince("2026-09-19T00:00:00Z")
	if err != nil {
		t.Fatalf("CountSince: %v", err)
	}
	if n != 2 {
		t.Errorf("CountSince today = %d, want 2", n)
	}
	n, err = r.CountBetween("2026-09-19T00:00:00Z", "2026-09-20T00:00:00Z")
	if err != nil || n != 2 {
		t.Errorf("CountBetween = %d, err=%v; want 2", n, err)
	}
}

func TestPageViewDailyUniques(t *testing.T) {
	d := seeded(t)
	r := NewPageViewRepo(d)
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	recordView(t, r, "/", "", "hash1", now)
	recordView(t, r, "/posts", "", "hash1", now.Add(time.Minute))
	recordView(t, r, "/about", "", "hash2", now.Add(2*time.Minute))
	recordView(t, r, "/", "", "hash3", now.AddDate(0, 0, -1))

	n, err := r.DailyUniqueCount("2026-09-19")
	if err != nil {
		t.Fatalf("DailyUniqueCount: %v", err)
	}
	if n != 2 {
		t.Errorf("dailies = %d, want 2", n)
	}
}

func TestPageViewTopPathsAndReferrers(t *testing.T) {
	d := seeded(t)
	r := NewPageViewRepo(d)
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	recordView(t, r, "/", "https://a.example", "x", now)
	recordView(t, r, "/", "https://a.example", "y", now)
	recordView(t, r, "/posts", "https://b.example", "z", now)

	paths, err := r.TopPaths("2026-09-01T00:00:00Z", 5)
	if err != nil {
		t.Fatalf("TopPaths: %v", err)
	}
	if len(paths) != 2 || paths[0].Path != "/" || paths[0].Count != 2 {
		t.Fatalf("paths = %+v", paths)
	}
	refs, err := r.TopReferrers("2026-09-01T00:00:00Z", 5)
	if err != nil {
		t.Fatalf("TopReferrers: %v", err)
	}
	if len(refs) != 2 || refs[0].Referrer != "https://a.example" || refs[0].Count != 2 {
		t.Fatalf("refs = %+v", refs)
	}

	recent, err := r.Recent(2)
	if err != nil || len(recent) != 2 {
		t.Fatalf("Recent = %d, err=%v", len(recent), err)
	}
}

func TestPageViewPruneAndClear(t *testing.T) {
	d := seeded(t)
	r := NewPageViewRepo(d)
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	recordView(t, r, "/old", "", "x", now.AddDate(0, 0, -100))
	recordView(t, r, "/new", "", "y", now)

	n, err := r.Prune("2026-09-01T00:00:00Z")
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned = %d, want 1", n)
	}
	if err := r.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if n, _ := r.CountSince("2020-01-01T00:00:00Z"); n != 0 {
		t.Errorf("after Clear = %d, want 0", n)
	}
}
