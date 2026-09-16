package web

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/icons"
	"github.com/mojoaar/johansenfoo/internal/markdown"
)

//go:embed templates/*.html
var templateFS embed.FS

type page struct {
	Profile        db.Profile
	Social         []db.SocialLink
	Projects       []db.Project
	Experience     []db.Experience
	Skills         []db.Skill
	HeroBio        template.HTML
	AboutPara1     template.HTML
	AboutPara2     template.HTML
	SiteName       string
	ThemeSlug      string
	ThemeCSS       template.CSS
	Meta           Meta
	StructuredData template.HTML
}

var templates = template.Must(
	template.New("").Funcs(template.FuncMap{
		"icon": func(name, class string) template.HTML {
			return icons.Inline(name, class)
		},
	}).ParseFS(templateFS, "templates/*.html"),
)

func renderPage(w http.ResponseWriter, name string, data page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func newPage(c *db.SiteContent, themeCSS string, meta Meta, structured template.HTML) page {
	return page{
		Profile:        c.Profile,
		Social:         c.Social,
		Projects:       c.Projects,
		Experience:     c.Experience,
		Skills:         c.Skills,
		HeroBio:        markdown.Render(c.Profile.HeroBio),
		AboutPara1:     markdown.Render(c.Profile.AboutPara1),
		AboutPara2:     markdown.Render(c.Profile.AboutPara2),
		ThemeSlug:      c.Theme.Slug,
		ThemeCSS:       template.CSS(themeCSS),
		Meta:           meta,
		StructuredData: structured,
	}
}
