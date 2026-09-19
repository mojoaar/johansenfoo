package db

import (
	"database/sql"
	"errors"

	"github.com/mojoaar/johansenfoo/internal/theme"
)

func SeedThemes(d *sql.DB, themes []theme.Theme) (int, error) {
	repo := NewThemeRepo(d)
	inserted := 0
	for _, t := range themes {
		_, err := repo.GetBySlug(t.Slug)
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrThemeNotFound) {
			return inserted, err
		}
		if _, err := repo.Create(&Theme{
			Slug:        t.Slug,
			Name:        t.Name,
			Description: t.Description,
			TokensBase:  t.Base,
			TokensLight: t.Light,
			TokensDark:  t.Dark,
		}); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}
