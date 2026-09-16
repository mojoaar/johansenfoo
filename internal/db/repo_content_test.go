package db

import (
	"database/sql"
	"testing"
)

func TestProjectCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateProject(&Project{
		Name: "zzz", URL: "https://example.com", Description: "desc",
		Icon: "globe", IsLink: true, URLLabel: "example.com", Sort: 99, Visible: true,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	p, err := r.Project(id)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Name != "zzz" || p.URL != "https://example.com" || !p.IsLink {
		t.Errorf("created = %+v", p)
	}

	p.Name = "yyy"
	p.URL = ""
	p.IsLink = false
	if err := r.UpdateProject(p); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	p, _ = r.Project(id)
	if p.Name != "yyy" || p.URL != "" || p.IsLink {
		t.Errorf("after update = %+v", p)
	}

	if err := r.DeleteProject(id); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := r.Project(id); err == nil {
		t.Error("project still readable after delete")
	}
}

func TestProjectUpdateKeepsNullURL(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	projects, err := r.Projects()
	if err != nil {
		t.Fatalf("Projects: %v", err)
	}
	var homelab Project
	for _, p := range projects {
		if p.Name == "homelab" {
			homelab = p
		}
	}
	if homelab.ID == 0 {
		t.Fatal("homelab project not found")
	}
	if homelab.URL != "" {
		t.Fatalf("homelab URL = %q, want empty", homelab.URL)
	}
	if err := r.UpdateProject(&homelab); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	var raw sql.NullString
	if err := d.QueryRow(`SELECT url FROM project WHERE id = ?`, homelab.ID).Scan(&raw); err != nil {
		t.Fatalf("raw select: %v", err)
	}
	if raw.Valid {
		t.Errorf("homelab url stored as %q, want NULL", raw.String)
	}
}

func TestProjectUpdateTogglesVisibility(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateProject(&Project{
		Name: "zzztoggle", URL: "https://toggle.example.com", Description: "toggle",
		Icon: "globe", IsLink: true, URLLabel: "toggle.example.com", Sort: 99, Visible: false,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	p, err := r.Project(id)
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Visible {
		t.Fatalf("precondition: Visible = true, want false")
	}

	p.Visible = true
	if err := r.UpdateProject(p); err != nil {
		t.Fatalf("UpdateProject(visible=true): %v", err)
	}
	if p, err = r.Project(id); err != nil {
		t.Fatalf("Project: %v", err)
	}
	if !p.Visible {
		t.Error("UpdateProject did not persist visible = true")
	}

	p.Visible = false
	if err := r.UpdateProject(p); err != nil {
		t.Fatalf("UpdateProject(visible=false): %v", err)
	}
	if p, err = r.Project(id); err != nil {
		t.Fatalf("Project: %v", err)
	}
	if p.Visible {
		t.Error("UpdateProject did not persist visible = false")
	}
}

func TestExperienceCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateExperience(&Experience{Years: "1990-1991", Role: "Tester", Company: "ACME", Icon: "briefcase", Sort: 99, Visible: true})
	if err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}
	e, err := r.ExperienceItem(id)
	if err != nil {
		t.Fatalf("ExperienceItem: %v", err)
	}
	if e.Role != "Tester" || e.Company != "ACME" {
		t.Errorf("created = %+v", e)
	}

	e.Role = "Senior Tester"
	if err := r.UpdateExperience(e); err != nil {
		t.Fatalf("UpdateExperience: %v", err)
	}
	e, _ = r.ExperienceItem(id)
	if e.Role != "Senior Tester" {
		t.Errorf("Role = %q", e.Role)
	}

	if err := r.DeleteExperience(id); err != nil {
		t.Fatalf("DeleteExperience: %v", err)
	}
	if _, err := r.ExperienceItem(id); err == nil {
		t.Error("experience still readable after delete")
	}
}

func TestSkillCRUD(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	id, err := r.CreateSkill(&Skill{Name: "Zig", Sort: 99, Visible: true})
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}
	s, err := r.Skill(id)
	if err != nil {
		t.Fatalf("Skill: %v", err)
	}
	if s.Name != "Zig" {
		t.Errorf("created = %+v", s)
	}

	s.Name = "Ziglang"
	s.Sort = 5
	if err := r.UpdateSkill(s); err != nil {
		t.Fatalf("UpdateSkill: %v", err)
	}
	s, _ = r.Skill(id)
	if s.Name != "Ziglang" || s.Sort != 5 {
		t.Errorf("after update = %+v", s)
	}

	if err := r.DeleteSkill(id); err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if _, err := r.Skill(id); err == nil {
		t.Error("skill still readable after delete")
	}
}

func TestExperienceListIncludesHidden(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	if _, err := r.CreateExperience(&Experience{
		Years: "x", Role: "Hidden Role", Company: "Hidden Co", Icon: "briefcase", Sort: 99, Visible: false,
	}); err != nil {
		t.Fatalf("CreateExperience: %v", err)
	}

	entries, err := r.Experience()
	if err != nil {
		t.Fatalf("Experience: %v", err)
	}
	var found *Experience
	for i := range entries {
		if entries[i].Company == "Hidden Co" {
			found = &entries[i]
		}
	}
	if found == nil {
		t.Fatal("hidden experience row is missing from Experience()")
	}
	if found.Visible {
		t.Error("hidden experience row reports Visible = true")
	}
	for _, e := range entries {
		if e.Company != "Hidden Co" && !e.Visible {
			t.Errorf("seeded experience row %q reports Visible = false", e.Company)
		}
	}
}

func TestSkillListIncludesHidden(t *testing.T) {
	d := seeded(t)
	r := NewContentRepo(d)

	if _, err := r.CreateSkill(&Skill{Name: "HiddenSkill", Sort: 99, Visible: false}); err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	skills, err := r.Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}
	var found *Skill
	for i := range skills {
		if skills[i].Name == "HiddenSkill" {
			found = &skills[i]
		}
	}
	if found == nil {
		t.Fatal("hidden skill row is missing from Skills()")
	}
	if found.Visible {
		t.Error("hidden skill row reports Visible = true")
	}
	for _, s := range skills {
		if s.Name != "HiddenSkill" && !s.Visible {
			t.Errorf("seeded skill %q reports Visible = false", s.Name)
		}
	}
}
