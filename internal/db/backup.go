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
	Posts      []Post            `json:"posts"`
	Tags       []Tag             `json:"tags"`
	Settings   map[string]string `json:"settings"`
}

func rfc3339OrNow(t time.Time) string {
	if t.IsZero() {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return t.UTC().Format(time.RFC3339)
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
	if social == nil {
		social = []SocialLink{}
	}
	if projects == nil {
		projects = []Project{}
	}
	if experience == nil {
		experience = []Experience{}
	}
	if skills == nil {
		skills = []Skill{}
	}
	posts, err := NewPostRepo(d).All()
	if err != nil {
		return nil, err
	}
	tags, err := NewPostRepo(d).Tags()
	if err != nil {
		return nil, err
	}
	if posts == nil {
		posts = []Post{}
	}
	if tags == nil {
		tags = []Tag{}
	}
	return &Snapshot{
		Version:    SnapshotVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Profile:    *profile,
		Social:     social,
		Projects:   projects,
		Experience: experience,
		Skills:     skills,
		Posts:      posts,
		Tags:       tags,
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
	if s.Social == nil || s.Projects == nil || s.Experience == nil || s.Skills == nil ||
		s.Posts == nil || s.Tags == nil {
		return errors.New("snapshot is missing one or more content sections")
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if slug, ok := s.Settings["active_theme"]; ok {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM theme WHERE slug = ?`, slug).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return errors.New("snapshot active_theme does not exist")
		}
	}

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
	if _, err := tx.Exec(`DELETE FROM post_tag`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM post`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tag`); err != nil {
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
	for _, tg := range s.Tags {
		if _, err := tx.Exec(
			`INSERT INTO tag (id, name, slug) VALUES (?, ?, ?)`,
			tg.ID, tg.Name, tg.Slug,
		); err != nil {
			return err
		}
	}
	tagID := make(map[string]int64, len(s.Tags))
	for _, tg := range s.Tags {
		tagID[tg.Slug] = tg.ID
	}
	for _, p := range s.Posts {
		var published any
		if p.PublishedAt != nil {
			published = p.PublishedAt.UTC().Format(time.RFC3339)
		}
		if _, err := tx.Exec(`INSERT INTO post
			(id, slug, title, summary, body_md, status, published_at, created_at, updated_at,
			 hero_image_url, hero_image_alt, seo_title, seo_description, og_image_url, canonical_url, noindex)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.ID, p.Slug, p.Title, p.Summary, p.BodyMD, p.Status, published,
			rfc3339OrNow(p.CreatedAt), rfc3339OrNow(p.UpdatedAt),
			p.HeroImageURL, p.HeroImageAlt, p.SEOTitle, p.SEODescription, p.OGImageURL,
			p.CanonicalURL, boolToInt(p.NoIndex)); err != nil {
			return err
		}
		for _, tg := range p.Tags {
			id := tg.ID
			if id == 0 {
				id = tagID[tg.Slug]
			}
			if id == 0 {
				continue
			}
			if _, err := tx.Exec(
				`INSERT OR IGNORE INTO post_tag (post_id, tag_id) VALUES (?, ?)`, p.ID, id); err != nil {
				return err
			}
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
