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
		SELECT id, years, role, company, icon, sort
		FROM experience WHERE visible = 1 ORDER BY sort, id`)
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
		SELECT id, name, sort FROM skill WHERE visible = 1 ORDER BY sort, id`)
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
