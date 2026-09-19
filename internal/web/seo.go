package web

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type Meta struct {
	Title       string
	Description string
	Canonical   string
	Robots      string
	OGType      string
	OGImage     string
	TwitterCard string
	TwitterSite string
	ThemeColor  string
}

type metaOverride struct {
	title       string
	description string
	image       string
	canonical   string
	noindex     bool
}

const defaultDescription = "Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years."

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func applyTitleTemplate(tmpl, title string) string {
	if tmpl == "" || !strings.Contains(tmpl, "%s") {
		return title
	}
	return strings.Replace(tmpl, "%s", title, 1)
}

func resolveMeta(c *db.SiteContent, route string, ov metaOverride) Meta {
	s := c.Settings
	page := c.PageSeo[route]

	title := page.Title
	if title == "" {
		if ov.title != "" {
			title = applyTitleTemplate(s["title_template"], ov.title)
		} else {
			title = s["site_title"]
		}
	}
	if title == "" {
		title = "Morten Johansen | johansen.foo"
	}

	description := firstNonEmpty(page.Description, ov.description, s["seo_description"], defaultDescription)
	image := firstNonEmpty(page.OGImageURL, ov.image, s["og_image_url"])

	base := s["canonical_base_url"]
	if base == "" {
		base = "https://johansen.foo"
	}
	canonical := firstNonEmpty(page.CanonicalURL, ov.canonical)
	if canonical == "" {
		if route == "" || route == "/" {
			canonical = base + "/"
		} else {
			canonical = base + route
		}
	}

	noindex := page.NoIndex || ov.noindex || s["noindex"] == "true"
	robots := "index, follow"
	if noindex {
		robots = "noindex, nofollow"
	}

	ogType := s["og_type"]
	if ogType == "" {
		ogType = "website"
	}
	card := s["twitter_card"]
	if card == "" {
		card = "summary"
	}

	return Meta{
		Title:       title,
		Description: description,
		Canonical:   canonical,
		Robots:      robots,
		OGType:      ogType,
		OGImage:     image,
		TwitterCard: card,
		TwitterSite: s["twitter_site"],
		ThemeColor:  themeColor(c),
	}
}

func themeColor(c *db.SiteContent) string {
	if v := themeFromRow(c.Theme).Dark["--bg"]; v != "" {
		return v
	}
	return "#0f1117"
}

func personSchema(c *db.SiteContent) template.HTML {
	sameAs := make([]string, 0, len(c.Social))
	for _, s := range c.Social {
		if !s.Visible {
			continue
		}
		if s.Platform != "github" {
			sameAs = append(sameAs, s.URL)
		}
	}
	for _, s := range c.Social {
		if !s.Visible {
			continue
		}
		if s.Platform == "github" {
			sameAs = append(sameAs, s.URL)
		}
	}

	knowsAbout := make([]string, 0, len(c.Skills))
	for _, s := range c.Skills {
		if !s.Visible {
			continue
		}
		knowsAbout = append(knowsAbout, s.Name)
	}

	payload := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "Person",
		"name":        c.Profile.Name,
		"url":         c.Settings["canonical_base_url"],
		"image":       c.Settings["og_image_url"],
		"jobTitle":    "Enterprise IT Leader",
		"description": c.Settings["seo_description"],
		"sameAs":      sameAs,
		"knowsAbout":  knowsAbout,
	}

	body, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(body) + `</script>`)
}

func blogPostSchema(c *db.SiteContent, p db.Post) template.HTML {
	base := c.Settings["canonical_base_url"]
	if base == "" {
		base = "https://johansen.foo"
	}
	published := p.CreatedAt
	if p.PublishedAt != nil {
		published = *p.PublishedAt
	}
	payload := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "BlogPosting",
		"headline":      p.Title,
		"description":   firstNonEmpty(p.SEODescription, p.Summary, c.Settings["seo_description"]),
		"url":           firstNonEmpty(p.CanonicalURL, base+"/posts/"+p.Slug),
		"image":         firstNonEmpty(p.OGImageURL, p.HeroImageURL, c.Settings["og_image_url"]),
		"datePublished": published.UTC().Format(time.RFC3339),
		"dateModified":  p.UpdatedAt.UTC().Format(time.RFC3339),
		"author":        map[string]string{"@type": "Person", "name": c.Profile.Name},
		"publisher":     map[string]string{"@type": "Person", "name": c.Profile.Name},
	}
	body, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return ""
	}
	return template.HTML(`<script type="application/ld+json">` + string(body) + `</script>`)
}

func robotsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		body := c.Settings["robots_txt"]
		if body == "" {
			body = "User-agent: *\nAllow: /\n"
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func sitemapHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		if c.Settings["sitemap_enabled"] == "false" {
			http.NotFound(w, r)
			return
		}
		base := c.Settings["canonical_base_url"]
		if base == "" {
			base = "https://johansen.foo"
		}

		var buf bytes.Buffer
		buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
		buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
		writeLoc := func(loc string) {
			buf.WriteString("  <url>\n    <loc>" + xmlEscape(loc) + "</loc>\n  </url>\n")
		}
		writeLoc(base + "/")
		if postsEnabled(c) {
			repo := db.NewPostRepo(d.DB)
			posts, err := repo.Published(1000, 0)
			if err != nil {
				http.Error(w, "storage error", http.StatusInternalServerError)
				return
			}
			tags, err := repo.Tags()
			if err != nil {
				http.Error(w, "storage error", http.StatusInternalServerError)
				return
			}
			writeLoc(base + "/posts")
			for _, p := range posts {
				writeLoc(base + "/posts/" + p.Slug)
			}
			for _, t := range tags {
				writeLoc(base + "/tags/" + t.Slug)
			}
		}
		buf.WriteString("</urlset>\n")

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	}
}
