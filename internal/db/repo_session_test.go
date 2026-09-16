package db

import (
	"errors"
	"testing"
	"time"
)

func TestSessionCreateAndGet(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	expires := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)

	if err := r.Create("abc123", expires); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s, err := r.Get("abc123")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.ID != "abc123" {
		t.Errorf("ID = %q, want %q", s.ID, "abc123")
	}
	if !s.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", s.ExpiresAt, expires)
	}
	if s.ExpiresAt.Location() != time.UTC {
		t.Errorf("ExpiresAt location = %v, want UTC", s.ExpiresAt.Location())
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestSessionGetMissing(t *testing.T) {
	d := seeded(t)
	if _, err := NewSessionRepo(d).Get("nope"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionDelete(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	if err := r.Create("gone", time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := r.Delete("gone"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.Get("gone"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("after Delete err = %v, want ErrSessionNotFound", err)
	}
}

func TestSessionDeleteExpired(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	past := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := r.Create("old", past); err != nil {
		t.Fatalf("Create old: %v", err)
	}
	if err := r.Create("live", future); err != nil {
		t.Fatalf("Create live: %v", err)
	}

	n, err := r.DeleteExpired()
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if n != 1 {
		t.Errorf("deleted = %d, want 1", n)
	}
	if _, err := r.Get("live"); err != nil {
		t.Errorf("live session should survive: %v", err)
	}
}

func TestSessionDeleteExpiredBoundary(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	const nowExpr = `strftime('%Y-%m-%dT%H:%M:%SZ','now')`

	for attempt := 0; attempt < 1000; attempt++ {
		if _, err := d.Exec(`DELETE FROM session`); err != nil {
			t.Fatalf("reset: %v", err)
		}
		if _, err := d.Exec(
			`INSERT INTO session (id, created_at, expires_at) VALUES
			 ('boundary', ` + nowExpr + `, ` + nowExpr + `),
			 ('future', ` + nowExpr + `, '2099-01-01T00:00:00Z')`,
		); err != nil {
			t.Fatalf("insert: %v", err)
		}

		var stored string
		if err := d.QueryRow(
			`SELECT expires_at FROM session WHERE id = 'boundary'`,
		).Scan(&stored); err != nil {
			t.Fatalf("read boundary: %v", err)
		}
		if len(stored) != 20 || stored[4] != '-' || stored[10] != 'T' ||
			stored[13] != ':' || stored[19] != 'Z' {
			t.Fatalf("stored expires_at = %q, want YYYY-MM-DDTHH:MM:SSZ", stored)
		}
		ts, err := time.Parse(time.RFC3339, stored)
		if err != nil {
			t.Fatalf("stored expires_at %q is not RFC 3339: %v", stored, err)
		}
		if ts.Location() != time.UTC {
			t.Fatalf("stored expires_at %q is not UTC", stored)
		}

		n, err := r.DeleteExpired()
		if err != nil {
			t.Fatalf("DeleteExpired: %v", err)
		}

		var after string
		if err := d.QueryRow(`SELECT ` + nowExpr).Scan(&after); err != nil {
			t.Fatalf("read now: %v", err)
		}
		if after != stored {
			continue
		}

		if _, err := r.Get("boundary"); err == nil {
			t.Fatal("DeleteExpired kept a session whose expires_at equals now; want the <= boundary deleted")
		}
		if _, err := r.Get("future"); err != nil {
			t.Fatalf("future session should survive: %v", err)
		}
		if n != 1 {
			t.Errorf("deleted = %d, want 1", n)
		}
		return
	}
	t.Fatal("could not observe the insert and the delete within the same second")
}

func TestSessionCreateNormalisesToUTC(t *testing.T) {
	d := seeded(t)
	r := NewSessionRepo(d)
	zone := time.FixedZone("UTC+05:30", 5*60*60+30*60)
	expires := time.Date(2026, 12, 31, 23, 59, 59, 0, zone)

	if err := r.Create("zoned", expires); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var stored string
	if err := d.QueryRow(
		`SELECT expires_at FROM session WHERE id = 'zoned'`,
	).Scan(&stored); err != nil {
		t.Fatalf("read expires_at: %v", err)
	}
	if want := expires.UTC().Format(time.RFC3339); stored != want {
		t.Errorf("stored expires_at = %q, want %q", stored, want)
	}
	if len(stored) != 20 || stored[19] != 'Z' {
		t.Errorf("stored expires_at = %q, want a 20-character UTC RFC 3339 value", stored)
	}

	s, err := r.Get("zoned")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.ExpiresAt.Location() != time.UTC {
		t.Errorf("ExpiresAt location = %v, want UTC", s.ExpiresAt.Location())
	}
	if !s.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want %v", s.ExpiresAt, expires)
	}
}
