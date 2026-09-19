package db

import (
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/mojoaar/johansenfoo/internal/theme"
)

var (
	ErrThemeNotFound  = errors.New("theme not found")
	ErrThemeProtected = errors.New("theme is protected")
)

type ThemeRepo struct{ db *sql.DB }

func NewThemeRepo(d *sql.DB) *ThemeRepo { return &ThemeRepo{db: d} }

const themeColumns = `id, slug, name, description, tokens_base, tokens_light, tokens_dark, sort`

func marshalTokens(t *Theme) (string, string, string, error) {
	base, err := json.Marshal(nonNilMap(t.TokensBase))
	if err != nil {
		return "", "", "", err
	}
	light, err := json.Marshal(nonNilMap(t.TokensLight))
	if err != nil {
		return "", "", "", err
	}
	dark, err := json.Marshal(nonNilMap(t.TokensDark))
	if err != nil {
		return "", "", "", err
	}
	return string(base), string(light), string(dark), nil
}

func nonNilMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

func scanTheme(row rowScanner) (*Theme, error) {
	var t Theme
	var baseRaw, lightRaw, darkRaw string
	if err := row.Scan(&t.ID, &t.Slug, &t.Name, &t.Description, &baseRaw, &lightRaw, &darkRaw, &t.Sort); err != nil {
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

func (r *ThemeRepo) GetBySlug(slug string) (*Theme, error) {
	t, err := scanTheme(r.db.QueryRow(`SELECT `+themeColumns+` FROM theme WHERE slug = ?`, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrThemeNotFound
	}
	return t, err
}

func (r *ThemeRepo) GetByID(id int64) (*Theme, error) {
	t, err := scanTheme(r.db.QueryRow(`SELECT `+themeColumns+` FROM theme WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrThemeNotFound
	}
	return t, err
}

func (r *ThemeRepo) List() ([]Theme, error) {
	rows, err := r.db.Query(`SELECT ` + themeColumns + ` FROM theme ORDER BY sort, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Theme
	for rows.Next() {
		t, err := scanTheme(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func themeToDomain(t *Theme) theme.Theme {
	return theme.Theme{
		Slug:  t.Slug,
		Base:  t.TokensBase,
		Light: t.TokensLight,
		Dark:  t.TokensDark,
	}
}

func (r *ThemeRepo) Create(t *Theme) (int64, error) {
	if err := theme.Validate(themeToDomain(t)); err != nil {
		return 0, err
	}
	base, light, dark, err := marshalTokens(t)
	if err != nil {
		return 0, err
	}
	res, err := r.db.Exec(
		`INSERT INTO theme (slug, name, description, tokens_base, tokens_light, tokens_dark, sort, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, `+nowExpr+`, `+nowExpr+`)`,
		t.Slug, t.Name, t.Description, base, light, dark, t.Sort,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *ThemeRepo) Update(t *Theme) error {
	existing, err := r.GetByID(t.ID)
	if err != nil {
		return err
	}
	if t.Slug != existing.Slug {
		active, err := r.ActiveSlug()
		if err != nil {
			return err
		}
		if existing.Slug == "johansen" || existing.Slug == active {
			return ErrThemeProtected
		}
	}
	merged := &Theme{
		ID:          t.ID,
		Slug:        t.Slug,
		Name:        t.Name,
		Description: t.Description,
		Sort:        t.Sort,
		TokensBase:  mergeTokens(existing.TokensBase, t.TokensBase),
		TokensLight: mergeTokens(existing.TokensLight, t.TokensLight),
		TokensDark:  mergeTokens(existing.TokensDark, t.TokensDark),
	}
	if err := theme.Validate(themeToDomain(merged)); err != nil {
		return err
	}
	base, light, dark, err := marshalTokens(merged)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`UPDATE theme SET slug = ?, name = ?, description = ?, tokens_base = ?, tokens_light = ?,
		   tokens_dark = ?, sort = ?, updated_at = `+nowExpr+` WHERE id = ?`,
		merged.Slug, merged.Name, merged.Description, base, light, dark, merged.Sort, merged.ID,
	)
	return err
}

func mergeTokens(existing, incoming map[string]string) map[string]string {
	out := make(map[string]string, len(existing)+len(incoming))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range incoming {
		out[k] = v
	}
	return out
}

func (r *ThemeRepo) Delete(id int64) error {
	t, err := r.GetByID(id)
	if err != nil {
		return err
	}
	if t.Slug == "johansen" {
		return ErrThemeProtected
	}
	active, err := r.ActiveSlug()
	if err != nil {
		return err
	}
	if t.Slug == active {
		return ErrThemeProtected
	}
	_, err = r.db.Exec(`DELETE FROM theme WHERE id = ?`, id)
	return err
}

func (r *ThemeRepo) ActiveSlug() (string, error) {
	v, err := NewSettingsRepo(r.db).Get("active_theme")
	if errors.Is(err, ErrSettingNotFound) {
		return "johansen", nil
	}
	if err != nil {
		return "", err
	}
	if v == "" {
		return "johansen", nil
	}
	return v, nil
}

func (r *ThemeRepo) SetActive(slug string) error {
	if _, err := r.GetBySlug(slug); err != nil {
		return err
	}
	return NewSettingsRepo(r.db).Set("active_theme", slug)
}
