package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminSkillsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "skills", "Skills")
		data.Skills = c.Skills
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_skills", data)
	}
}

func adminSkillCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		s := &db.Skill{
			Name:    r.FormValue("name"),
			Sort:    formInt(r, "sort"),
			Visible: r.FormValue("visible") != "",
		}
		if _, err := db.NewContentRepo(d.DB).CreateSkill(s); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}

func adminSkillUpdateHandler(d Deps) http.HandlerFunc {
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
		s := &db.Skill{
			ID:      id,
			Name:    r.FormValue("name"),
			Sort:    formInt(r, "sort"),
			Visible: r.FormValue("visible") != "",
		}
		if err := db.NewContentRepo(d.DB).UpdateSkill(s); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}

func adminSkillDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteSkill(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/skills?saved=1", http.StatusSeeOther)
	}
}
