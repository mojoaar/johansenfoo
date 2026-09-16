package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func seeded(t *testing.T) *sql.DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := Migrate(d); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return d
}

func count(t *testing.T, d *sql.DB, table string) int {
	t.Helper()
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestSeedCounts(t *testing.T) {
	d := seeded(t)

	cases := []struct {
		table string
		want  int
	}{
		{"profile", 1},
		{"social_link", 4},
		{"project", 9},
		{"experience", 8},
		{"skill", 30},
		{"theme", 1},
	}
	for _, c := range cases {
		if got := count(t, d, c.table); got != c.want {
			t.Errorf("%s count = %d, want %d", c.table, got, c.want)
		}
	}
}

func TestSeedProfileMatchesCurrentSite(t *testing.T) {
	d := seeded(t)

	var name, handle, location, dob string
	err := d.QueryRow(
		`SELECT name, handle, location, dob FROM profile WHERE id = 1`,
	).Scan(&name, &handle, &location, &dob)
	if err != nil {
		t.Fatalf("profile: %v", err)
	}

	if name != "Morten Johansen" {
		t.Errorf("name = %q", name)
	}
	if handle != "mojoaar" {
		t.Errorf("handle = %q", handle)
	}
	if location != "Denmark" {
		t.Errorf("location = %q", location)
	}
	if dob != "1980-08-13" {
		t.Errorf("dob = %q", dob)
	}
}

func TestSeedProjectOrderAndHomelabIsNotALink(t *testing.T) {
	d := seeded(t)

	rows, err := d.Query(`
		SELECT name, is_link, url_label FROM project ORDER BY sort`)
	if err != nil {
		t.Fatalf("projects: %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name, urlLabel string
		var isLink int
		if err := rows.Scan(&name, &isLink, &urlLabel); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
		if name == "homelab" {
			if isLink != 0 {
				t.Error("homelab is_link = 1, want 0")
			}
			if urlLabel != "self-hosted // private" {
				t.Errorf("homelab url_label = %q", urlLabel)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"atlascmdb", "clutch", "echo", "homelab", "icloud-mailflow",
		"ignite", "kanzo", "krypt", "mindmatrix",
	}
	if len(names) != len(want) {
		t.Fatalf("got %d projects, want %d", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("project[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestSeedSocialLinksAndSkills(t *testing.T) {
	d := seeded(t)

	rows, err := d.Query(`SELECT platform FROM social_link ORDER BY sort`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var platforms []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		platforms = append(platforms, p)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	want := []string{"bluesky", "linkedin", "mastodon", "github"}
	for i, w := range want {
		if i >= len(platforms) || platforms[i] != w {
			t.Fatalf("social platforms = %v, want %v", platforms, want)
		}
	}

	var firstName, lastName string
	if err := d.QueryRow(`SELECT name FROM skill ORDER BY sort LIMIT 1`).Scan(&firstName); err != nil {
		t.Fatal(err)
	}
	if firstName != "ITSM" {
		t.Errorf("first skill = %q, want ITSM", firstName)
	}
	if err := d.QueryRow(`SELECT name FROM skill ORDER BY sort DESC LIMIT 1`).Scan(&lastName); err != nil {
		t.Fatal(err)
	}
	if lastName != "Presenting" {
		t.Errorf("last skill = %q, want Presenting", lastName)
	}
}

func TestSeedSettingsDefaults(t *testing.T) {
	d := seeded(t)

	want := map[string]string{
		"posts_enabled":        "true",
		"active_theme":         "johansen",
		"stats_enabled":        "true",
		"stats_retention_days": "90",
		"timezone":             "Europe/Copenhagen",
		"site_title":           "Morten Johansen | johansen.foo",
		"seo_description":      "Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years. Global ops leader, open-source tinkerer, and automation enthusiast.",
	}
	for key, wantVal := range want {
		var got string
		if err := d.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&got); err != nil {
			t.Errorf("setting %q missing: %v", key, err)
			continue
		}
		if got != wantVal {
			t.Errorf("setting %q = %q, want %q", key, got, wantVal)
		}
	}
}
