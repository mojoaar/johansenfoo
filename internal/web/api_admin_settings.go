package web

import (
	"net/http"
	"strconv"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type postsSettingRequest struct {
	Enabled *bool `json:"enabled"`
}

type themeSettingRequest struct {
	Slug string `json:"slug"`
}

func apiAdminPostsSettingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req postsSettingRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Enabled == nil {
			writeAPIError(w, http.StatusBadRequest, "enabled is required")
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set("posts_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"enabled": *req.Enabled})
	}
}

func apiAdminThemeSettingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req themeSettingRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Slug == "" {
			writeAPIError(w, http.StatusBadRequest, "slug is required")
			return
		}
		if _, err := db.NewThemeRepo(d.DB).GetBySlug(req.Slug); err != nil {
			writeAPIError(w, http.StatusBadRequest, "unknown theme")
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set("active_theme", req.Slug); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"slug": req.Slug})
	}
}
