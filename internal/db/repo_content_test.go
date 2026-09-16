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
