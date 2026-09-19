package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminProfileGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := db.NewProfileRepo(d.DB).Get()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func apiAdminProfilePutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p db.Profile
		if !decodeJSON(w, r, &p) {
			return
		}
		if err := validateProfile(&p); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if err := repo.Update(&p); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.Get()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func validateProfile(p *db.Profile) error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}
