package web

import (
	"errors"
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

func validateAPITheme(t *db.Theme) error {
	return theme.Validate(theme.Theme{
		Slug:  t.Slug,
		Base:  t.TokensBase,
		Light: t.TokensLight,
		Dark:  t.TokensDark,
	})
}

func mergeThemeTokens(existing, incoming map[string]string) map[string]string {
	out := make(map[string]string, len(existing)+len(incoming))
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range incoming {
		out[k] = v
	}
	return out
}

func apiAdminThemesListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		themes, err := db.NewThemeRepo(d.DB).List()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, themes)
	}
}

func apiAdminThemesGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		th, err := db.NewThemeRepo(d.DB).GetByID(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "theme not found")
			return
		}
		writeJSON(w, http.StatusOK, th)
	}
}

func apiAdminThemesCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var t db.Theme
		if !decodeJSON(w, r, &t) {
			return
		}
		if t.Slug == "" || t.Name == "" {
			writeAPIError(w, http.StatusBadRequest, "slug and name are required")
			return
		}
		if err := validateAPITheme(&t); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		repo := db.NewThemeRepo(d.DB)
		id, err := repo.Create(&t)
		if err != nil {
			if db.IsUniqueViolation(err) {
				writeAPIError(w, http.StatusConflict, "slug already in use")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		created, err := repo.GetByID(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func apiAdminThemesUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewThemeRepo(d.DB)
		existing, err := repo.GetByID(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "theme not found")
			return
		}
		var t db.Theme
		if !decodeJSON(w, r, &t) {
			return
		}
		merged := *existing
		if t.Slug != "" {
			merged.Slug = t.Slug
		}
		if t.Name != "" {
			merged.Name = t.Name
		}
		if hasBody(t.Description) {
			merged.Description = t.Description
		}
		merged.TokensBase = mergeThemeTokens(existing.TokensBase, t.TokensBase)
		merged.TokensLight = mergeThemeTokens(existing.TokensLight, t.TokensLight)
		merged.TokensDark = mergeThemeTokens(existing.TokensDark, t.TokensDark)
		if err := validateAPITheme(&merged); err != nil {
			writeAPIError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := repo.Update(&merged); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.GetByID(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func hasBody(s string) bool {
	return s != ""
}

func apiAdminThemesDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		err := db.NewThemeRepo(d.DB).Delete(id)
		if errors.Is(err, db.ErrThemeProtected) {
			writeAPIError(w, http.StatusConflict, "the base and active themes cannot be deleted")
			return
		}
		if errors.Is(err, db.ErrThemeNotFound) {
			writeAPIError(w, http.StatusNotFound, "theme not found")
			return
		}
		if err != nil {
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

func apiAdminThemesActivateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewThemeRepo(d.DB)
		th, err := repo.GetByID(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "theme not found")
			return
		}
		if err := repo.SetActive(th.Slug); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"active": th.Slug})
	}
}
