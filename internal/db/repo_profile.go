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
		SELECT id, platform, url, label, sort
		FROM social_link WHERE visible = 1 ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SocialLink
	for rows.Next() {
		var s SocialLink
		if err := rows.Scan(&s.ID, &s.Platform, &s.URL, &s.Label, &s.Sort); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
