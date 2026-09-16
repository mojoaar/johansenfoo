package db

import (
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/theme"
)

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

func TestContentRepoOrdersByIDAscendingWhenSortTies(t *testing.T) {
	d := seeded(t)

	if _, err := d.Exec(`CREATE INDEX idx_skill_sort_desc ON skill(sort DESC)`); err != nil {
		t.Fatalf("create index: %v", err)
	}
	if _, err := d.Exec(
		`INSERT INTO skill (name, sort, visible) VALUES ('tie-a', 100, 1), ('tie-b', 100, 1)`,
	); err != nil {
		t.Fatalf("insert tied rows: %v", err)
	}

	skills, err := NewContentRepo(d).Skills()
	if err != nil {
		t.Fatalf("Skills: %v", err)
	}

	var got []int64
	for _, s := range skills {
		if s.Name == "tie-a" || s.Name == "tie-b" {
			got = append(got, s.ID)
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d tied skills, want 2", len(got))
	}
	if got[0] >= got[1] {
		t.Errorf("equal sort rows returned ids %v, want ascending", got)
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

func TestSeededThemeMatchesBuiltin(t *testing.T) {
	d := seeded(t)

	got, err := NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}

	want := theme.Johansen()

	for _, c := range []struct {
		label string
		got   map[string]string
		want  map[string]string
	}{
		{"base", got.TokensBase, want.Base},
		{"light", got.TokensLight, want.Light},
		{"dark", got.TokensDark, want.Dark},
	} {
		if len(c.got) != len(c.want) {
			t.Errorf("%s: got %d tokens, want %d", c.label, len(c.got), len(c.want))
		}
		for k, v := range c.want {
			if c.got[k] != v {
				t.Errorf("%s %s = %q, want %q", c.label, k, c.got[k], v)
			}
		}
	}
}

func TestJohansenThemeSeedIsIdempotent(t *testing.T) {
	d := seeded(t)

	const sentinel = "2000-01-01T00:00:00Z"
	if _, err := d.Exec(
		`UPDATE theme SET created_at = ? WHERE slug = 'johansen'`, sentinel,
	); err != nil {
		t.Fatalf("set created_at sentinel: %v", err)
	}

	if _, err := d.Exec(`DELETE FROM schema_migrations WHERE version = 3`); err != nil {
		t.Fatalf("delete migration record: %v", err)
	}
	if err := Migrate(d); err != nil {
		t.Fatalf("re-run Migrate: %v", err)
	}

	if got := count(t, d, "theme"); got != 1 {
		t.Fatalf("theme count = %d, want 1", got)
	}

	got, err := NewThemeRepo(d).GetBySlug("johansen")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	want := theme.Johansen()
	for k, v := range want.Dark {
		if got.TokensDark[k] != v {
			t.Errorf("dark %s = %q, want %q", k, got.TokensDark[k], v)
		}
	}

	var createdAt, updatedAt string
	if err := d.QueryRow(
		`SELECT created_at, updated_at FROM theme WHERE slug = 'johansen'`,
	).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("read timestamps: %v", err)
	}
	if createdAt != sentinel {
		t.Errorf("created_at = %q, want %q preserved across re-migrate", createdAt, sentinel)
	}
	for _, ts := range []string{createdAt, updatedAt} {
		parsed, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			t.Errorf("timestamp %q is not RFC 3339: %v", ts, err)
			continue
		}
		if parsed.Location() != time.UTC {
			t.Errorf("timestamp %q location = %v, want UTC", ts, parsed.Location())
		}
	}
}
