package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type crudRepo[T any] struct {
	list     func() ([]T, error)
	get      func(int64) (*T, error)
	create   func(*T) (int64, error)
	update   func(*T) error
	delete   func(int64) error
	setID    func(*T, int64)
	validate func(*T) error
}

func apiCRUDHandler[T any](d Deps, repo crudRepo[T], resource string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if chi.URLParam(r, "id") != "" {
				id, ok := apiID(w, r)
				if !ok {
					return
				}
				item, err := repo.get(id)
				if err != nil {
					writeAPIError(w, http.StatusNotFound, resource+" not found")
					return
				}
				writeJSON(w, http.StatusOK, item)
				return
			}
			items, err := repo.list()
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			var item T
			if !decodeJSON(w, r, &item) {
				return
			}
			if repo.validate != nil {
				if err := repo.validate(&item); err != nil {
					writeAPIError(w, http.StatusBadRequest, err.Error())
					return
				}
			}
			id, err := repo.create(&item)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			created, err := repo.get(id)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusCreated, created)
		case http.MethodPut:
			id, ok := apiID(w, r)
			if !ok {
				return
			}
			if _, err := repo.get(id); err != nil {
				writeAPIError(w, http.StatusNotFound, resource+" not found")
				return
			}
			var item T
			if !decodeJSON(w, r, &item) {
				return
			}
			if repo.validate != nil {
				if err := repo.validate(&item); err != nil {
					writeAPIError(w, http.StatusBadRequest, err.Error())
					return
				}
			}
			repo.setID(&item, id)
			if err := repo.update(&item); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "save failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			updated, err := repo.get(id)
			if err != nil {
				writeAPIError(w, http.StatusInternalServerError, "storage error")
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			id, ok := apiID(w, r)
			if !ok {
				return
			}
			if _, err := repo.get(id); err != nil {
				writeAPIError(w, http.StatusNotFound, resource+" not found")
				return
			}
			if err := repo.delete(id); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "delete failed")
				return
			}
			if err := d.Content.Reload(); err != nil {
				writeAPIError(w, http.StatusInternalServerError, "reload failed")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}

func projectCRUD(d Deps) crudRepo[db.Project] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Project]{
		list:   r.Projects,
		get:    r.Project,
		create: r.CreateProject,
		update: r.UpdateProject,
		delete: r.DeleteProject,
		setID:  func(p *db.Project, id int64) { p.ID = id },
		validate: func(p *db.Project) error {
			if strings.TrimSpace(p.Name) == "" {
				return errors.New("name is required")
			}
			return nil
		},
	}
}

func experienceCRUD(d Deps) crudRepo[db.Experience] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Experience]{
		list:   r.Experience,
		get:    r.ExperienceItem,
		create: r.CreateExperience,
		update: r.UpdateExperience,
		delete: r.DeleteExperience,
		setID:  func(e *db.Experience, id int64) { e.ID = id },
		validate: func(e *db.Experience) error {
			if strings.TrimSpace(e.Role) == "" {
				return errors.New("role is required")
			}
			return nil
		},
	}
}

func skillCRUD(d Deps) crudRepo[db.Skill] {
	r := db.NewContentRepo(d.DB)
	return crudRepo[db.Skill]{
		list:   r.Skills,
		get:    r.Skill,
		create: r.CreateSkill,
		update: r.UpdateSkill,
		delete: r.DeleteSkill,
		setID:  func(s *db.Skill, id int64) { s.ID = id },
		validate: func(s *db.Skill) error {
			if strings.TrimSpace(s.Name) == "" {
				return errors.New("name is required")
			}
			return nil
		},
	}
}
