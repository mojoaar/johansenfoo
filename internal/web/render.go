package web

import (
	"bytes"
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
	ThemeSlug      string
	ThemeCSS       template.CSS
	Meta           Meta
	StructuredData template.HTML
	Page           string
	Section        string
	Title          string
	CSRF           string
	Flash          string
	Error          string
	Version        string
	APIKey         string
	HasAPIKey      bool
	IsNew          bool
	Project        db.Project
	Item           db.Experience
	Skill          db.Skill
}

var templates = template.Must(
	template.New("").Funcs(template.FuncMap{
		"icon": func(name, class string) template.HTML {
			return icons.Inline(name, class)
		},
	}).ParseFS(templateFS, "templates/*.html"),
)

func renderPage(w http.ResponseWriter, name string, data page) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func newPage(c *db.SiteContent, themeCSS string, meta Meta, structured template.HTML) page {
	return page{
		Profile:        c.Profile,
		Social:         visibleSocial(c.Social),
		Projects:       visibleProjects(c.Projects),
		Experience:     visibleExperience(c.Experience),
		Skills:         visibleSkills(c.Skills),
		HeroBio:        markdown.Render(c.Profile.HeroBio),
		AboutPara1:     markdown.Render(c.Profile.AboutPara1),
		AboutPara2:     markdown.Render(c.Profile.AboutPara2),
		ThemeSlug:      c.Theme.Slug,
		ThemeCSS:       template.CSS(themeCSS),
		Meta:           meta,
		StructuredData: structured,
	}
}

func visibleSocial(links []db.SocialLink) []db.SocialLink {
	out := make([]db.SocialLink, 0, len(links))
	for _, l := range links {
		if l.Visible {
			out = append(out, l)
		}
	}
	return out
}

func visibleProjects(projects []db.Project) []db.Project {
	out := make([]db.Project, 0, len(projects))
	for _, p := range projects {
		if p.Visible {
			out = append(out, p)
		}
	}
	return out
}

func visibleExperience(entries []db.Experience) []db.Experience {
	out := make([]db.Experience, 0, len(entries))
	for _, e := range entries {
		if e.Visible {
			out = append(out, e)
		}
	}
	return out
}

func visibleSkills(skills []db.Skill) []db.Skill {
	out := make([]db.Skill, 0, len(skills))
	for _, s := range skills {
		if s.Visible {
			out = append(out, s)
		}
	}
	return out
}

func NewAdminPage(d Deps, r *http.Request, section, title string) page {
	p := page{
		Page:    section,
		Section: section,
		Title:   title,
		Version: d.Version,
	}
	if c := d.Content.Current(); c != nil {
		p.ThemeSlug = c.Theme.Slug
	}
	return p
}

func renderAdmin(w http.ResponseWriter, r *http.Request, name string, data page, status ...int) {
	data.CSRF = ensureCSRFCookie(w, r)
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(status) > 0 {
		w.WriteHeader(status[0])
	}
	_, _ = w.Write(buf.Bytes())
}
