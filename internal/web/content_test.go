package web

import "testing"

func TestContentStoreCurrentReturnsSnapshot(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	c := s.Current()
	if c == nil {
		t.Fatal("Current() returned nil")
	}
	if c.Profile.Name != "Morten Johansen" {
		t.Errorf("Profile.Name = %q", c.Profile.Name)
	}
	if len(c.Projects) != 9 {
		t.Errorf("len(Projects) = %d, want 9", len(c.Projects))
	}
}

func TestContentStoreReloadPicksUpWrites(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	if s.Current().Profile.Name == "Changed Name" {
		t.Fatal("precondition failed: name already changed")
	}

	if _, err := d.Exec(`UPDATE profile SET name = 'Changed Name' WHERE id = 1`); err != nil {
		t.Fatalf("update: %v", err)
	}
	if s.Current().Profile.Name == "Changed Name" {
		t.Fatal("Current() reflected the write before Reload()")
	}
	if err := s.Reload(); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := s.Current().Profile.Name; got != "Changed Name" {
		t.Errorf("after Reload Profile.Name = %q, want %q", got, "Changed Name")
	}
}

func TestContentStoreReloadErrorLeavesPreviousSnapshot(t *testing.T) {
	d := newTestDB(t)
	s, err := NewContentStore(d)
	if err != nil {
		t.Fatalf("NewContentStore: %v", err)
	}
	before := s.Current()

	if _, err := d.Exec(`DELETE FROM profile`); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := s.Reload(); err == nil {
		t.Fatal("Reload() succeeded with no profile row, want error")
	}
	if s.Current() != before {
		t.Error("failed Reload() replaced the snapshot")
	}
}
