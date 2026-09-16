package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestPruneSessionsRemovesOnlyExpired(t *testing.T) {
	d := newTestDB(t)
	r := db.NewSessionRepo(d)
	if err := r.Create("expired", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Create expired: %v", err)
	}
	if err := r.Create("live", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create live: %v", err)
	}

	n, err := pruneSessions(d)
	if err != nil {
		t.Fatalf("pruneSessions: %v", err)
	}
	if n != 1 {
		t.Errorf("pruned = %d, want 1", n)
	}
	if _, err := r.Get("live"); err != nil {
		t.Errorf("live session was pruned: %v", err)
	}
}

func TestStartSessionPrunerDeletesExpiredOnSchedule(t *testing.T) {
	d := newTestDB(t)
	r := db.NewSessionRepo(d)
	if err := r.Create("expired", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Create expired: %v", err)
	}
	if err := r.Create("live", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create live: %v", err)
	}

	stop := startSessionPruner(d, 10*time.Millisecond)
	defer stop()

	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := r.Get("expired"); errors.Is(err, db.ErrSessionNotFound) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expired session was not pruned within the deadline")
		}
		time.Sleep(5 * time.Millisecond)
	}

	if _, err := r.Get("live"); err != nil {
		t.Errorf("live session was pruned: %v", err)
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	rec := doGet(t, h, "/health")
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q", got)
	}
	if got := rec.Header().Get("Referrer-Policy"); got != "strict-origin-when-cross-origin" {
		t.Errorf("Referrer-Policy = %q", got)
	}
}

func TestStaticDirectoryListingIsRejected(t *testing.T) {
	d := newTestDB(t)
	store, _ := NewContentStore(d)
	h := newAuthTestHandler(t, d, store)

	for _, path := range []string{"/static/", "/static/fonts/"} {
		rec := doGet(t, h, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", path, rec.Code)
		}
	}
	rec := doGet(t, h, "/static/style.css")
	if rec.Code != http.StatusOK {
		t.Errorf("GET /static/style.css = %d, want 200", rec.Code)
	}
}

func doGet(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}
