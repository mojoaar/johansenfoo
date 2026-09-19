package db

import (
	"database/sql"
	"errors"
	"time"
)

const SnapshotVersion = 1

var secretSettingKeys = map[string]bool{
	"admin_password_hash": true,
	"api_key":             true,
}

type Snapshot struct {
	Version    int               `json:"version"`
	ExportedAt string            `json:"exported_at"`
	Profile    Profile           `json:"profile"`
	Social     []SocialLink      `json:"social"`
	Projects   []Project         `json:"projects"`
	Experience []Experience      `json:"experience"`
	Skills     []Skill           `json:"skills"`
	Settings   map[string]string `json:"settings"`
}

func Export(d *sql.DB) (*Snapshot, error) {
	profile, err := NewProfileRepo(d).Get()
	if err != nil {
		return nil, err
	}
	social, err := NewProfileRepo(d).SocialLinks()
	if err != nil {
		return nil, err
	}
	content := NewContentRepo(d)
	projects, err := content.Projects()
	if err != nil {
		return nil, err
	}
	experience, err := content.Experience()
	if err != nil {
		return nil, err
	}
	skills, err := content.Skills()
	if err != nil {
		return nil, err
	}
	settings, err := NewSettingsRepo(d).All()
	if err != nil {
		return nil, err
	}
	for k := range settings {
		if secretSettingKeys[k] {
			delete(settings, k)
		}
	}
	return &Snapshot{
		Version:    SnapshotVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profile:    *profile,
		Social:     social,
		Projects:   projects,
		Experience: experience,
		Skills:     skills,
		Settings:   settings,
	}, nil
}

func Import(d *sql.DB, s *Snapshot) error {
	if s == nil || s.Version != SnapshotVersion {
		return errors.New("unsupported snapshot version")
	}
	if s.Profile.Name == "" {
		return errors.New("snapshot profile name is required")
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM social_link`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM project`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM experience`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM skill`); err != nil {
		return err
	}

	p := s.Profile
	if _, err := tx.Exec(
		`UPDATE profile SET name = ?, handle = ?, location = ?, dob = ?, tagline = ?,
		   hero_bio = ?, bio = ?, about_para_1 = ?, about_para_2 = ?, avatar = ?,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE id = 1`,
		p.Name, p.Handle, p.Location, p.DOB, p.Tagline,
		p.HeroBio, p.Bio, p.AboutPara1, p.AboutPara2, p.Avatar,
	); err != nil {
		return err
	}

	for _, l := range s.Social {
		if _, err := tx.Exec(
			`INSERT INTO social_link (id, platform, url, label, sort, visible) VALUES (?, ?, ?, ?, ?, ?)`,
			l.ID, l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible),
		); err != nil {
			return err
		}
	}
	for _, pr := range s.Projects {
		if _, err := tx.Exec(
			`INSERT INTO project (id, name, url, description, icon, is_link, url_label, sort, visible)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			pr.ID, pr.Name, nullableURL(pr.URL), pr.Description, pr.Icon,
			boolToInt(pr.IsLink), pr.URLLabel, pr.Sort, boolToInt(pr.Visible),
		); err != nil {
			return err
		}
	}
	for _, e := range s.Experience {
		if _, err := tx.Exec(
			`INSERT INTO experience (id, years, role, company, icon, sort, visible) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible),
		); err != nil {
			return err
		}
	}
	for _, sk := range s.Skills {
		if _, err := tx.Exec(
			`INSERT INTO skill (id, name, sort, visible) VALUES (?, ?, ?, ?)`,
			sk.ID, sk.Name, sk.Sort, boolToInt(sk.Visible),
		); err != nil {
			return err
		}
	}
	for k, v := range s.Settings {
		if secretSettingKeys[k] {
			continue
		}
		if _, err := tx.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			k, v,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
