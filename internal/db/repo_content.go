package db

import "database/sql"

type ContentRepo struct{ db *sql.DB }

func NewContentRepo(d *sql.DB) *ContentRepo { return &ContentRepo{db: d} }

func (r *ContentRepo) Projects() ([]Project, error) {
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(url, ''), description, icon, is_link, url_label, sort
		FROM project WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Description,
			&p.Icon, &p.IsLink, &p.URLLabel, &p.Sort); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Experience() ([]Experience, error) {
	rows, err := r.db.Query(`
		SELECT id, years, role, company, icon, sort
		FROM experience WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Experience
	for rows.Next() {
		var e Experience
		if err := rows.Scan(&e.ID, &e.Years, &e.Role, &e.Company, &e.Icon, &e.Sort); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Skills() ([]Skill, error) {
	rows, err := r.db.Query(`
		SELECT id, name, sort FROM skill WHERE visible = 1 ORDER BY sort`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		var s Skill
		if err := rows.Scan(&s.ID, &s.Name, &s.Sort); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
