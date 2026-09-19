package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminStatsVisitorsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "7d"
		}
		stats, err := collectVisitors(d, period)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func apiAdminStatsClearHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := db.NewPageViewRepo(d.DB).Clear(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "clear failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

type statsSettingRequest struct {
	Enabled       *bool `json:"enabled"`
	RetentionDays *int  `json:"retention_days"`
}

func apiAdminStatsSettingsPutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req statsSettingRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		settings := db.NewSettingsRepo(d.DB)
		if req.Enabled != nil {
			if err := settings.Set("stats_enabled", strconv.FormatBool(*req.Enabled)); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
		}
		if req.RetentionDays != nil {
			if *req.RetentionDays <= 0 {
				writeAPIError(w, http.StatusBadRequest, "retention_days must be positive")
				return
			}
			if err := settings.Set("stats_retention_days", strconv.Itoa(*req.RetentionDays)); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		out, err := db.NewSettingsRepo(d.DB).All()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled":        out["stats_enabled"] != "false",
			"retention_days": out["stats_retention_days"],
		})
	}
}

func adminStatsPostHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		settings := db.NewSettingsRepo(d.DB)
		enabled := "false"
		if r.FormValue("stats_enabled") != "" {
			enabled = "true"
		}
		if err := settings.Set("stats_enabled", enabled); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if days := strings.TrimSpace(r.FormValue("stats_retention_days")); days != "" {
			if _, err := strconv.Atoi(days); err != nil {
				http.Error(w, "retention must be a number", http.StatusBadRequest)
				return
			}
			if err := settings.Set("stats_retention_days", days); err != nil {
				http.Error(w, "save failed", http.StatusInternalServerError)
				return
			}
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin?saved=1", http.StatusSeeOther)
	}
}

func apiAdminStatsSystemHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, collectRuntime(d))
	}
}
