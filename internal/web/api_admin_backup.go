package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminExportHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap, err := db.Export(d.DB)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "export failed")
			return
		}
		writeJSON(w, http.StatusOK, snap)
	}
}

func apiAdminImportHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var snap db.Snapshot
		if !decodeJSON(w, r, &snap) {
			return
		}
		if err := db.Import(d.DB, &snap); err != nil {
			writeAPIError(w, http.StatusBadRequest, "invalid snapshot")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
