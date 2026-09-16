package db

import "testing"

func TestProfileRepoUpdate(t *testing.T) {
	d := seeded(t)
	r := NewProfileRepo(d)

	p, err := r.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	p.Name = "New Name"
	p.Tagline = "// new tagline"
	if err := r.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := r.Get()
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Name != "New Name" || got.Tagline != "// new tagline" {
		t.Errorf("name=%q tagline=%q", got.Name, got.Tagline)
	}
	if got.Bio == "" {
		t.Error("Update cleared the bio column")
	}
}

func TestSocialLinkCRUD(t *testing.T) {
	d := seeded(t)
	r := NewProfileRepo(d)

	id, err := r.CreateSocialLink(&SocialLink{Platform: "example", URL: "https://example.com", Label: "Example", Sort: 99})
	if err != nil {
		t.Fatalf("CreateSocialLink: %v", err)
	}
	if id == 0 {
		t.Fatal("CreateSocialLink returned id 0")
	}

	l, err := r.SocialLink(id)
	if err != nil {
		t.Fatalf("SocialLink: %v", err)
	}
	if l.Platform != "example" || l.URL != "https://example.com" {
		t.Errorf("created link = %+v", l)
	}

	l.Label = "Renamed"
	if err := r.UpdateSocialLink(l); err != nil {
		t.Fatalf("UpdateSocialLink: %v", err)
	}
	l, _ = r.SocialLink(id)
	if l.Label != "Renamed" {
		t.Errorf("Label = %q, want %q", l.Label, "Renamed")
	}

	if err := r.DeleteSocialLink(id); err != nil {
		t.Fatalf("DeleteSocialLink: %v", err)
	}
	if _, err := r.SocialLink(id); err == nil {
		t.Error("link still readable after delete")
	}
}
