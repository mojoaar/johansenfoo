package db

import "database/sql"

type ProfileRepo struct{ db *sql.DB }

func NewProfileRepo(d *sql.DB) *ProfileRepo { return &ProfileRepo{db: d} }

func (r *ProfileRepo) Get() (*Profile, error) {
	var p Profile
	err := r.db.QueryRow(`
		SELECT name, handle, location, dob, tagline, hero_bio,
		       about_para_1, about_para_2, avatar, bio
		FROM profile WHERE id = 1`).
		Scan(&p.Name, &p.Handle, &p.Location, &p.DOB, &p.Tagline,
			&p.HeroBio, &p.AboutPara1, &p.AboutPara2, &p.Avatar, &p.Bio)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProfileRepo) SocialLinks() ([]SocialLink, error) {
	rows, err := r.db.Query(`
		SELECT id, platform, url, label, sort, visible
		FROM social_link ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SocialLink
	for rows.Next() {
		var s SocialLink
		var visible int
		if err := rows.Scan(&s.ID, &s.Platform, &s.URL, &s.Label, &s.Sort, &visible); err != nil {
			return nil, err
		}
		s.Visible = visible != 0
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ProfileRepo) Update(p *Profile) error {
	_, err := r.db.Exec(
		`UPDATE profile SET
		   name = ?, handle = ?, location = ?, dob = ?, tagline = ?,
		   hero_bio = ?, bio = ?, about_para_1 = ?, about_para_2 = ?, avatar = ?,
		   updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE id = 1`,
		p.Name, p.Handle, p.Location, p.DOB, p.Tagline,
		p.HeroBio, p.Bio, p.AboutPara1, p.AboutPara2, p.Avatar,
	)
	return err
}

func (r *ProfileRepo) SocialLink(id int64) (*SocialLink, error) {
	var l SocialLink
	var visible int
	err := r.db.QueryRow(
		`SELECT id, platform, url, label, sort, visible FROM social_link WHERE id = ?`, id,
	).Scan(&l.ID, &l.Platform, &l.URL, &l.Label, &l.Sort, &visible)
	if err != nil {
		return nil, err
	}
	l.Visible = visible != 0
	return &l, nil
}

func (r *ProfileRepo) CreateSocialLink(l *SocialLink) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO social_link (platform, url, label, sort, visible) VALUES (?, ?, ?, ?, ?)`,
		l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ProfileRepo) UpdateSocialLink(l *SocialLink) error {
	_, err := r.db.Exec(
		`UPDATE social_link SET platform = ?, url = ?, label = ?, sort = ?, visible = ? WHERE id = ?`,
		l.Platform, l.URL, l.Label, l.Sort, boolToInt(l.Visible), l.ID,
	)
	return err
}

func (r *ProfileRepo) DeleteSocialLink(id int64) error {
	_, err := r.db.Exec(`DELETE FROM social_link WHERE id = ?`, id)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
