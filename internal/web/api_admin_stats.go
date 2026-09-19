package web

import (
	"net/http"

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
