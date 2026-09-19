package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type seoRequest struct {
	SiteTitle        string        `json:"site_title"`
	TitleTemplate    string        `json:"title_template"`
	SEODescription   string        `json:"seo_description"`
	OGImageURL       string        `json:"og_image_url"`
	OGType           string        `json:"og_type"`
	TwitterCard      string        `json:"twitter_card"`
	TwitterSite      string        `json:"twitter_site"`
	CanonicalBaseURL string        `json:"canonical_base_url"`
	NoIndex          bool          `json:"noindex"`
	SitemapEnabled   bool          `json:"sitemap_enabled"`
	RobotsTxt        string        `json:"robots_txt"`
	Pages            *[]db.PageSeo `json:"pages"`
}

func currentSEO(d Deps) (seoRequest, error) {
	c := d.Content.Current()
	pages, err := db.NewPageSeoRepo(d.DB).List()
	if err != nil {
		return seoRequest{}, err
	}
	if pages == nil {
		pages = []db.PageSeo{}
	}
	s := c.Settings
	return seoRequest{
		SiteTitle:        s["site_title"],
		TitleTemplate:    s["title_template"],
		SEODescription:   s["seo_description"],
		OGImageURL:       s["og_image_url"],
		OGType:           s["og_type"],
		TwitterCard:      s["twitter_card"],
		TwitterSite:      s["twitter_site"],
		CanonicalBaseURL: s["canonical_base_url"],
		NoIndex:          s["noindex"] == "true",
		SitemapEnabled:   s["sitemap_enabled"] != "false",
		RobotsTxt:        s["robots_txt"],
		Pages:            &pages,
	}, nil
}

func apiAdminSeoGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out, err := currentSEO(d)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func apiAdminSeoPutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req seoRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Pages == nil {
			writeAPIError(w, http.StatusBadRequest, "pages is required")
			return
		}

		if err := db.NewPageSeoRepo(d.DB).ReplaceAll(*req.Pages); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid pages")
			return
		}

		repo := db.NewSettingsRepo(d.DB)
		stringSettings := map[string]string{
			"site_title":         req.SiteTitle,
			"title_template":     req.TitleTemplate,
			"seo_description":    req.SEODescription,
			"og_image_url":       req.OGImageURL,
			"og_type":            req.OGType,
			"twitter_card":       req.TwitterCard,
			"twitter_site":       req.TwitterSite,
			"canonical_base_url": req.CanonicalBaseURL,
			"robots_txt":         req.RobotsTxt,
		}
		for key, value := range stringSettings {
			if err := repo.Set(key, value); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
		}
		for key, value := range map[string]bool{"noindex": req.NoIndex, "sitemap_enabled": req.SitemapEnabled} {
			setting := "false"
			if value {
				setting = "true"
			}
			if err := repo.Set(key, setting); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
		}

		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		out, err := currentSEO(d)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
