package db

import "testing"

func TestProfileRepoGet(t *testing.T) {
	d := seeded(t)

	p, err := NewProfileRepo(d).Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "Morten Johansen" {
		t.Errorf("Name = %q", p.Name)
	}
	if p.HeroBio == "" || p.AboutPara1 == "" {
		t.Error("bio fields are empty")
	}
}

func TestContentRepoOrdering(t *testing.T) {
	d := seeded(t)
	repo := NewContentRepo(d)

	projects, err := repo.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	if len(projects) != 9 {
		t.Fatalf("got %d projects, want 9", len(projects))
	}
	if projects[3].Name != "homelab" || projects[3].IsLink {
		t.Errorf("projects[3] = %+v, want non-link homelab", projects[3])
	}

	exp, err := repo.Experience()
	if err != nil {
		t.Fatalf("Experience: %v", err)
	}
	if len(exp) != 8 {
		t.Fatalf("got %d experience rows, want 8", len(exp))
	}
	if exp[0].Company != "JYSK" || exp[7].Company != "Jydske Dragonregiment" {
		t.Errorf("experience order wrong: first=%q last=%q", exp[0].Company, exp[7].Company)
	}

	skills, err := repo.Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	if len(skills) != 30 {
		t.Fatalf("got %d skills, want 30", len(skills))
	}
	if skills[0].Name != "ITSM" || skills[29].Name != "Presenting" {
		t.Errorf("skill order wrong: first=%q last=%q", skills[0].Name, skills[29].Name)
	}
}

func TestThemeRepoGetBySlug(t *testing.T) {
	d := seeded(t)

	th, err := NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if th.Name != "Johansen" {
		t.Errorf("Name = %q", th.Name)
	}
	if th.TokensBase == nil || th.TokensLight == nil || th.TokensDark == nil {
		t.Errorf("token maps not parsed: %+v", th)
	}
}

func TestSettingsRepo(t *testing.T) {
	d := seeded(t)
	repo := NewSettingsRepo(d)

	got, err := repo.Get("active_theme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "johansen" {
		t.Errorf("active_theme = %q, want johansen", got)
	}

	on, err := repo.GetBool("posts_enabled")
	if err != nil {
		t.Fatalf("GetBool: %v", err)
	}
	if !on {
		t.Error("posts_enabled = false, want true")
	}
}

func TestSettingsRepoMissingKeyErrors(t *testing.T) {
	d := seeded(t)
	if _, err := NewSettingsRepo(d).Get("does_not_exist"); err == nil {
		t.Fatal("Get on missing key succeeded, want error")
	}
}
