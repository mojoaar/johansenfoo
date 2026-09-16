package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminExperienceGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "experience", "Experience")
		data.Experience = c.Experience
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_experience", data)
	}
}

func adminExperienceNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "experience", "New experience entry")
		data.IsNew = true
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_experience_form", data)
	}
}

func adminExperienceEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		e, err := db.NewContentRepo(d.DB).ExperienceItem(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		data := NewAdminPage(d, r, "experience", "Edit experience entry")
		data.Item = *e
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_experience_form", data)
	}
}

func formExperience(r *http.Request) *db.Experience {
	return &db.Experience{
		Years:   r.FormValue("years"),
		Role:    r.FormValue("role"),
		Company: r.FormValue("company"),
		Icon:    r.FormValue("icon"),
		Sort:    formInt(r, "sort"),
		Visible: r.FormValue("visible") != "",
	}
}

func adminExperienceCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if _, err := db.NewContentRepo(d.DB).CreateExperience(formExperience(r)); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}

func adminExperienceUpdateHandler(d Deps) http.HandlerFunc {
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
		e := formExperience(r)
		e.ID = id
		if err := db.NewContentRepo(d.DB).UpdateExperience(e); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}

func adminExperienceDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteExperience(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/experience?saved=1", http.StatusSeeOther)
	}
}
