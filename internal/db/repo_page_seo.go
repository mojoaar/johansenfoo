package db

import (
	"database/sql"
	"errors"
)

var ErrPageSeoNotFound = errors.New("page seo not found")

type PageSeoRepo struct{ db *sql.DB }

func NewPageSeoRepo(d *sql.DB) *PageSeoRepo { return &PageSeoRepo{db: d} }

const pageSeoColumns = `id, route, title, description, og_image_url, canonical_url, noindex`

func scanPageSeo(row rowScanner) (*PageSeo, error) {
	var p PageSeo
	var noindex int
	if err := row.Scan(&p.ID, &p.Route, &p.Title, &p.Description, &p.OGImageURL, &p.CanonicalURL, &noindex); err != nil {
		return nil, err
	}
	p.NoIndex = noindex != 0
	return &p, nil
}

func (r *PageSeoRepo) Get(route string) (*PageSeo, error) {
	p, err := scanPageSeo(r.db.QueryRow(`SELECT `+pageSeoColumns+` FROM page_seo WHERE route = ?`, route))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPageSeoNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *PageSeoRepo) List() ([]PageSeo, error) {
	rows, err := r.db.Query(`SELECT ` + pageSeoColumns + ` FROM page_seo ORDER BY route`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PageSeo
	for rows.Next() {
		p, err := scanPageSeo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *PageSeoRepo) Upsert(p *PageSeo) error {
	_, err := r.db.Exec(
		`INSERT INTO page_seo (route, title, description, og_image_url, canonical_url, noindex)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(route) DO UPDATE SET
		   title = excluded.title,
		   description = excluded.description,
		   og_image_url = excluded.og_image_url,
		   canonical_url = excluded.canonical_url,
		   noindex = excluded.noindex`,
		p.Route, p.Title, p.Description, p.OGImageURL, p.CanonicalURL, boolToInt(p.NoIndex),
	)
	return err
}

func (r *PageSeoRepo) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM page_seo WHERE id = ?`, id)
	return err
}

func (r *PageSeoRepo) ReplaceAll(pages []PageSeo) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM page_seo`); err != nil {
		return err
	}
	for _, p := range pages {
		if p.Route == "" {
			return errors.New("page seo route is required")
		}
		if _, err := tx.Exec(
			`INSERT INTO page_seo (route, title, description, og_image_url, canonical_url, noindex)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			p.Route, p.Title, p.Description, p.OGImageURL, p.CanonicalURL, boolToInt(p.NoIndex),
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
