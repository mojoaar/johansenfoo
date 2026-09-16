package web

import (
	"encoding/json"
	"html/template"

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
	ThemeColor  string
}

const defaultDescription = "Morten Johansen - Building and running complex infrastructure & cloud environments for 18+ years."

func resolveMeta(c *db.SiteContent, route string) Meta {
	s := c.Settings

	title := s["site_title"]
	if title == "" {
		title = "Morten Johansen | johansen.foo"
	}
	description := s["seo_description"]
	if description == "" {
		description = defaultDescription
	}
	base := s["canonical_base_url"]
	if base == "" {
		base = "https://johansen.foo"
	}
	robots := "index, follow"
	if s["noindex"] == "true" {
		robots = "noindex, nofollow"
	}

	canonical := base + "/"
	if route != "/" {
		canonical = base + route
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
		OGImage:     s["og_image_url"],
		TwitterCard: card,
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
		if s.Platform != "github" {
			sameAs = append(sameAs, s.URL)
		}
	}
	for _, s := range c.Social {
		if s.Platform == "github" {
			sameAs = append(sameAs, s.URL)
		}
	}

	knowsAbout := make([]string, 0, len(c.Skills))
	for _, s := range c.Skills {
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
