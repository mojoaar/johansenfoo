package db

import (
	"testing"
)

func TestExportOmitsSecrets(t *testing.T) {
	d := seeded(t)
	if err := NewSettingsRepo(d).Set("api_key", "super-secret"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	if err := NewSettingsRepo(d).Set("admin_password_hash", "$2a$hash"); err != nil {
		t.Fatalf("Set password hash: %v", err)
	}

	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if _, ok := snap.Settings["api_key"]; ok {
		t.Error("api_key leaked into export")
	}
	if _, ok := snap.Settings["admin_password_hash"]; ok {
		t.Error("admin_password_hash leaked into export")
	}
	if snap.Settings["active_theme"] != "johansen" {
		t.Errorf("active_theme = %q, want johansen", snap.Settings["active_theme"])
	}
	if snap.Profile.Name == "" || len(snap.Projects) != 9 || len(snap.Skills) != 30 {
		t.Errorf("incomplete snapshot: name=%q projects=%d skills=%d",
			snap.Profile.Name, len(snap.Projects), len(snap.Skills))
	}
}

func TestImportRoundTrip(t *testing.T) {
	d := seeded(t)
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if _, err := NewContentRepo(d).CreateProject(&Project{Name: "temporary", Visible: true}); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if err := NewProfileRepo(d).Update(&Profile{Name: "Changed"}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := Import(d, snap); err != nil {
		t.Fatalf("Import: %v", err)
	}

	profile, err := NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if profile.Name != "Morten Johansen" {
		t.Errorf("name = %q, want Morten Johansen", profile.Name)
	}
	projects, err := NewContentRepo(d).Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 9 {
		t.Errorf("projects = %d, want 9", len(projects))
	}
}

func TestImportRollsBackOnFailure(t *testing.T) {
	d := seeded(t)
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	snap.Social = []SocialLink{
		{ID: 1, Platform: "a", URL: "https://a.example", Label: "A"},
		{ID: 1, Platform: "b", URL: "https://b.example", Label: "B"},
	}

	if err := Import(d, snap); err == nil {
		t.Fatal("Import succeeded with a duplicate primary key")
	}

	links, err := NewProfileRepo(d).SocialLinks()
	if err != nil {
		t.Fatalf("SocialLinks: %v", err)
	}
	if len(links) != 4 {
		t.Fatalf("social = %d, want 4 (transaction did not roll back)", len(links))
	}
	if links[0].Platform != "bluesky" {
		t.Errorf("first platform = %q, want bluesky", links[0].Platform)
	}
}

func TestImportRejectsMissingProfile(t *testing.T) {
	d := seeded(t)
	if err := Import(d, &Snapshot{Version: SnapshotVersion}); err == nil {
		t.Fatal("Import accepted a snapshot with no profile name")
	}
}

func TestImportRejectsUnknownTheme(t *testing.T) {
	d := seeded(t)
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	snap.Settings["active_theme"] = "does-not-exist"

	if err := Import(d, snap); err == nil {
		t.Fatal("Import accepted an active_theme that does not exist")
	}

	got, err := NewSettingsRepo(d).Get("active_theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "johansen" {
		t.Errorf("active_theme = %q, want johansen (import was not rejected atomically)", got)
	}
}

func TestImportRejectsMissingSections(t *testing.T) {
	d := seeded(t)
	partial := &Snapshot{Version: SnapshotVersion, Profile: Profile{Name: "x"}}

	if err := Import(d, partial); err == nil {
		t.Fatal("Import accepted a snapshot with omitted content sections")
	}

	projects, err := NewContentRepo(d).Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 9 {
		t.Errorf("projects = %d, want 9 (partial import destroyed content)", len(projects))
	}
}

func TestImportIgnoresSecrets(t *testing.T) {
	d := seeded(t)
	settings := NewSettingsRepo(d)
	if err := settings.Set("api_key", "keep-me"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	if err := settings.Set("admin_password_hash", "keep-hash"); err != nil {
		t.Fatalf("Set hash: %v", err)
	}
	snap, err := Export(d)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	snap.Settings["api_key"] = "overwrite"
	snap.Settings["admin_password_hash"] = "overwrite"

	if err := Import(d, snap); err != nil {
		t.Fatalf("Import: %v", err)
	}

	key, err := settings.Get("api_key")
	if err != nil || key != "keep-me" {
		t.Errorf("api_key = %q, err=%v; want keep-me", key, err)
	}
	hash, err := settings.Get("admin_password_hash")
	if err != nil || hash != "keep-hash" {
		t.Errorf("admin_password_hash = %q, err=%v; want keep-hash", hash, err)
	}
}
