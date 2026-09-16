package web

import (
	"database/sql"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

func LoadContent(d *sql.DB) (*db.SiteContent, error) {
	profileRepo := db.NewProfileRepo(d)
	profile, err := profileRepo.Get()
	if err != nil {
		return nil, err
	}
	social, err := profileRepo.SocialLinks()
	if err != nil {
		return nil, err
	}
	content := db.NewContentRepo(d)
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

	settings, err := db.NewSettingsRepo(d).All()
	if err != nil {
		return nil, err
	}

	slug := settings["active_theme"]
	row, err := db.NewThemeRepo(d).GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	return &db.SiteContent{
		Profile:    *profile,
		Social:     social,
		Projects:   projects,
		Experience: experience,
		Skills:     skills,
		Theme:      *row,
		Settings:   settings,
	}, nil
}

func themeFromRow(row db.Theme) theme.Theme {
	return theme.Theme{
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		Base:        row.TokensBase,
		Light:       row.TokensLight,
		Dark:        row.TokensDark,
	}
}

func landingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		themeCSS := theme.CSS(themeFromRow(c.Theme))
		meta := resolveMeta(c, "/")
		data := newPage(c, themeCSS, meta, personSchema(c))
		renderPage(w, "base", data)
	}
}
