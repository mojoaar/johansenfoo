package db

import "database/sql"

type ContentRepo struct{ db *sql.DB }

func NewContentRepo(d *sql.DB) *ContentRepo { return &ContentRepo{db: d} }

func (r *ContentRepo) Projects() ([]Project, error) {
	rows, err := r.db.Query(`
		SELECT id, name, COALESCE(url, ''), description, icon, is_link, url_label, sort, visible
		FROM project ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var visible int
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Description,
			&p.Icon, &p.IsLink, &p.URLLabel, &p.Sort, &visible); err != nil {
			return nil, err
		}
		p.Visible = visible != 0
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Project(id int64) (*Project, error) {
	var p Project
	var visible int
	var url sql.NullString
	err := r.db.QueryRow(
		`SELECT id, name, url, description, icon, is_link, url_label, sort, visible
		 FROM project WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &url, &p.Description, &p.Icon, &p.IsLink, &p.URLLabel, &p.Sort, &visible)
	if err != nil {
		return nil, err
	}
	p.URL = url.String
	p.Visible = visible != 0
	return &p, nil
}

func (r *ContentRepo) CreateProject(p *Project) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO project (name, url, description, icon, is_link, url_label, sort, visible)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, nullableURL(p.URL), p.Description, p.Icon, boolToInt(p.IsLink), p.URLLabel, p.Sort, boolToInt(p.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateProject(p *Project) error {
	_, err := r.db.Exec(
		`UPDATE project SET name = ?, url = ?, description = ?, icon = ?, is_link = ?, url_label = ?, sort = ?, visible = ?
		 WHERE id = ?`,
		p.Name, nullableURL(p.URL), p.Description, p.Icon, boolToInt(p.IsLink), p.URLLabel, p.Sort, boolToInt(p.Visible), p.ID,
	)
	return err
}

func (r *ContentRepo) DeleteProject(id int64) error {
	_, err := r.db.Exec(`DELETE FROM project WHERE id = ?`, id)
	return err
}

func nullableURL(u string) any {
	if u == "" {
		return nil
	}
	return u
}

func (r *ContentRepo) Experience() ([]Experience, error) {
	rows, err := r.db.Query(`
		SELECT id, years, role, company, icon, sort, visible
		FROM experience ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Experience
	for rows.Next() {
		var e Experience
		var visible int
		if err := rows.Scan(&e.ID, &e.Years, &e.Role, &e.Company, &e.Icon, &e.Sort, &visible); err != nil {
			return nil, err
		}
		e.Visible = visible != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *ContentRepo) Skills() ([]Skill, error) {
	rows, err := r.db.Query(`
		SELECT id, name, sort, visible FROM skill ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Skill
	for rows.Next() {
		var s Skill
		var visible int
		if err := rows.Scan(&s.ID, &s.Name, &s.Sort, &visible); err != nil {
			return nil, err
		}
		s.Visible = visible != 0
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ContentRepo) ExperienceItem(id int64) (*Experience, error) {
	var e Experience
	var visible int
	err := r.db.QueryRow(
		`SELECT id, years, role, company, icon, sort, visible FROM experience WHERE id = ?`, id,
	).Scan(&e.ID, &e.Years, &e.Role, &e.Company, &e.Icon, &e.Sort, &visible)
	if err != nil {
		return nil, err
	}
	e.Visible = visible != 0
	return &e, nil
}

func (r *ContentRepo) CreateExperience(e *Experience) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO experience (years, role, company, icon, sort, visible) VALUES (?, ?, ?, ?, ?, ?)`,
		e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateExperience(e *Experience) error {
	_, err := r.db.Exec(
		`UPDATE experience SET years = ?, role = ?, company = ?, icon = ?, sort = ?, visible = ? WHERE id = ?`,
		e.Years, e.Role, e.Company, e.Icon, e.Sort, boolToInt(e.Visible), e.ID,
	)
	return err
}

func (r *ContentRepo) DeleteExperience(id int64) error {
	_, err := r.db.Exec(`DELETE FROM experience WHERE id = ?`, id)
	return err
}

func (r *ContentRepo) Skill(id int64) (*Skill, error) {
	var s Skill
	var visible int
	err := r.db.QueryRow(
		`SELECT id, name, sort, visible FROM skill WHERE id = ?`, id,
	).Scan(&s.ID, &s.Name, &s.Sort, &visible)
	if err != nil {
		return nil, err
	}
	s.Visible = visible != 0
	return &s, nil
}

func (r *ContentRepo) CreateSkill(s *Skill) (int64, error) {
	res, err := r.db.Exec(
		`INSERT INTO skill (name, sort, visible) VALUES (?, ?, ?)`,
		s.Name, s.Sort, boolToInt(s.Visible),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ContentRepo) UpdateSkill(s *Skill) error {
	_, err := r.db.Exec(
		`UPDATE skill SET name = ?, sort = ?, visible = ? WHERE id = ?`,
		s.Name, s.Sort, boolToInt(s.Visible), s.ID,
	)
	return err
}

func (r *ContentRepo) DeleteSkill(id int64) error {
	_, err := r.db.Exec(`DELETE FROM skill WHERE id = ?`, id)
	return err
}
