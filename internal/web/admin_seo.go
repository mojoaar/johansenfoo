package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
)

var seoStringKeys = []string{
	"site_title", "title_template", "seo_description", "og_image_url",
	"og_type", "twitter_card", "twitter_site", "canonical_base_url", "robots_txt",
}

func adminSeoGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		pages, err := db.NewPageSeoRepo(d.DB).List()
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		data := NewAdminPage(d, r, "seo", "SEO")
		data.Settings = c.Settings
		data.PageSeoList = pages
		renderAdmin(w, r, "admin_seo", data)
	}
}

func adminSeoPostHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		repo := db.NewSettingsRepo(d.DB)
		for _, key := range seoStringKeys {
			if err := repo.Set(key, r.FormValue(key)); err != nil {
				http.Error(w, "save failed", http.StatusInternalServerError)
				return
			}
		}
		for key, formField := range map[string]string{"noindex": "noindex", "sitemap_enabled": "sitemap_enabled"} {
			value := "false"
			if r.FormValue(formField) != "" {
				value = "true"
			}
			if err := repo.Set(key, value); err != nil {
				http.Error(w, "save failed", http.StatusInternalServerError)
				return
			}
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/seo?saved=1", http.StatusSeeOther)
	}
}

func formPageSeo(r *http.Request) *db.PageSeo {
	return &db.PageSeo{
		Route:        strings.TrimSpace(r.FormValue("route")),
		Title:        r.FormValue("title"),
		Description:  r.FormValue("description"),
		OGImageURL:   r.FormValue("og_image_url"),
		CanonicalURL: r.FormValue("canonical_url"),
		NoIndex:      r.FormValue("noindex") != "",
	}
}

func adminSeoPageUpsertHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		page := formPageSeo(r)
		if page.Route == "" {
			http.Error(w, "route is required", http.StatusBadRequest)
			return
		}
		if err := db.NewPageSeoRepo(d.DB).Upsert(page); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/seo?saved=1", http.StatusSeeOther)
	}
}

func adminSeoPageDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewPageSeoRepo(d.DB).Delete(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/seo?saved=1", http.StatusSeeOther)
	}
}
