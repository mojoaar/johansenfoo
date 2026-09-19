package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

func adminThemesGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		themes, err := db.NewThemeRepo(d.DB).List()
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		active, err := db.NewThemeRepo(d.DB).ActiveSlug()
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		data := NewAdminPage(d, r, "themes", "Themes")
		data.Themes = themes
		data.ActiveTheme = active
		renderAdmin(w, r, "admin_themes", data)
	}
}

func adminThemeNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		base, err := db.NewThemeRepo(d.DB).GetBySlug("johansen")
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		start := *base
		start.ID = 0
		start.Slug = ""
		start.Name = ""
		start.Description = ""
		data := NewAdminPage(d, r, "themes", "New theme")
		data.IsNew = true
		data.Theme = start
		renderAdmin(w, r, "admin_theme_form", data)
	}
}

func adminThemeEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		th, err := db.NewThemeRepo(d.DB).GetByID(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		data := NewAdminPage(d, r, "themes", "Edit theme")
		data.Theme = *th
		renderAdmin(w, r, "admin_theme_form", data)
	}
}

func parseTokens(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}
	var tokens map[string]string
	if err := json.Unmarshal([]byte(raw), &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

func themeFromForm(r *http.Request, id int64) (*db.Theme, error) {
	base, err := parseTokens(r.FormValue("tokens_base"))
	if err != nil {
		return nil, errors.New("tokens_base is not valid JSON")
	}
	light, err := parseTokens(r.FormValue("tokens_light"))
	if err != nil {
		return nil, errors.New("tokens_light is not valid JSON")
	}
	dark, err := parseTokens(r.FormValue("tokens_dark"))
	if err != nil {
		return nil, errors.New("tokens_dark is not valid JSON")
	}
	th := &db.Theme{
		ID:          id,
		Slug:        strings.TrimSpace(r.FormValue("slug")),
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: r.FormValue("description"),
		TokensBase:  base,
		TokensLight: light,
		TokensDark:  dark,
	}
	if th.Slug == "" || th.Name == "" {
		return nil, errors.New("slug and name are required")
	}
	if err := theme.Validate(theme.Theme{
		Slug:  th.Slug,
		Base:  th.TokensBase,
		Light: th.TokensLight,
		Dark:  th.TokensDark,
	}); err != nil {
		return nil, err
	}
	return th, nil
}

func adminThemeCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		th, err := themeFromForm(r, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := db.NewThemeRepo(d.DB).Create(th); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/themes?saved=1", http.StatusSeeOther)
	}
}

func adminThemeUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		th, err := themeFromForm(r, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := db.NewThemeRepo(d.DB).Update(th); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/themes?saved=1", http.StatusSeeOther)
	}
}

func adminThemeDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		err = db.NewThemeRepo(d.DB).Delete(id)
		if errors.Is(err, db.ErrThemeProtected) {
			http.Error(w, "the base and active themes cannot be deleted", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/themes?saved=1", http.StatusSeeOther)
	}
}

func adminThemeActivateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		th, err := db.NewThemeRepo(d.DB).GetByID(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := db.NewThemeRepo(d.DB).SetActive(th.Slug); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/themes?saved=1", http.StatusSeeOther)
	}
}
