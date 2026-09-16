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
