package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func apiAdminSocialListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		links, err := db.NewProfileRepo(d.DB).SocialLinks()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, links)
	}
}

func apiAdminSocialCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var l db.SocialLink
		if !decodeJSON(w, r, &l) {
			return
		}
		if err := validateSocial(&l); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		repo := db.NewProfileRepo(d.DB)
		id, err := repo.CreateSocialLink(&l)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		created, err := repo.SocialLink(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func apiAdminSocialUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if _, err := repo.SocialLink(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "social link not found")
			return
		}
		var l db.SocialLink
		if !decodeJSON(w, r, &l) {
			return
		}
		if err := validateSocial(&l); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		l.ID = id
		if err := repo.UpdateSocialLink(&l); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.SocialLink(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func apiAdminSocialDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewProfileRepo(d.DB)
		if _, err := repo.SocialLink(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "social link not found")
			return
		}
		if err := repo.DeleteSocialLink(id); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "delete failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func validateSocial(l *db.SocialLink) error {
	if strings.TrimSpace(l.Platform) == "" {
		return errors.New("platform is required")
	}
	if strings.TrimSpace(l.URL) == "" {
		return errors.New("url is required")
	}
	return nil
}
