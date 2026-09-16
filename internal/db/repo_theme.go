package db

import (
	"database/sql"
	"encoding/json"
)

type ThemeRepo struct{ db *sql.DB }

func NewThemeRepo(d *sql.DB) *ThemeRepo { return &ThemeRepo{db: d} }

func (r *ThemeRepo) GetBySlug(slug string) (*Theme, error) {
	var t Theme
	var baseRaw, lightRaw, darkRaw string

	err := r.db.QueryRow(`
		SELECT id, slug, name, description, tokens_base, tokens_light, tokens_dark
		FROM theme WHERE slug = ?`, slug).
		Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &baseRaw, &lightRaw, &darkRaw)
	if err != nil {
		return nil, err
	}

	for _, f := range []struct {
		raw  string
		into *map[string]string
	}{
		{baseRaw, &t.TokensBase},
		{lightRaw, &t.TokensLight},
		{darkRaw, &t.TokensDark},
	} {
		if err := json.Unmarshal([]byte(f.raw), f.into); err != nil {
			return nil, err
		}
	}
	return &t, nil
}
