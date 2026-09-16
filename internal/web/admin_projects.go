package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminProjectsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "projects", "Projects")
		data.Projects = c.Projects
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_projects", data)
	}
}

func adminProjectNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "projects", "New project")
		data.IsNew = true
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_project_form", data)
	}
}

func adminProjectEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		p, err := db.NewContentRepo(d.DB).Project(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		data := NewAdminPage(d, r, "projects", "Edit project")
		data.Project = *p
		data.ThemeSlug = d.Content.Current().Theme.Slug
		renderAdmin(w, r, "admin_project_form", data)
	}
}

func formProject(r *http.Request) *db.Project {
	return &db.Project{
		Name:        r.FormValue("name"),
		URL:         r.FormValue("url"),
		Description: r.FormValue("description"),
		Icon:        r.FormValue("icon"),
		IsLink:      r.FormValue("url") != "",
		URLLabel:    r.FormValue("url_label"),
		Sort:        formInt(r, "sort"),
		Visible:     r.FormValue("visible") != "",
	}
}

func adminProjectCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if _, err := db.NewContentRepo(d.DB).CreateProject(formProject(r)); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}

func adminProjectUpdateHandler(d Deps) http.HandlerFunc {
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
		p := formProject(r)
		p.ID = id
		if err := db.NewContentRepo(d.DB).UpdateProject(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}

func adminProjectDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewContentRepo(d.DB).DeleteProject(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/projects?saved=1", http.StatusSeeOther)
	}
}
