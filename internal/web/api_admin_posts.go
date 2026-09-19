package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

type apiTags []db.Tag

func (t *apiTags) UnmarshalJSON(data []byte) error {
	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		out := make(apiTags, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			out = append(out, db.Tag{Name: name, Slug: db.Slugify(name)})
		}
		*t = out
		return nil
	}
	var objs []db.Tag
	if err := json.Unmarshal(data, &objs); err != nil {
		return err
	}
	*t = objs
	return nil
}

type apiPostRequest struct {
	db.Post
	Tags apiTags `json:"tags"`
}

func (req *apiPostRequest) post() db.Post {
	p := req.Post
	p.Tags = []db.Tag(req.Tags)
	return p
}

func validateAPIPost(p *db.Post) bool {
	if p.Status != "" && p.Status != "draft" && p.Status != "published" {
		return false
	}
	if p.Slug == "" {
		p.Slug = db.Slugify(p.Title)
	}
	return p.Slug != "" && p.Title != ""
}

func apiAdminPostsListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		posts, err := db.NewPostRepo(d.DB).All()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, posts)
	}
}

func apiAdminPostsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		p, err := db.NewPostRepo(d.DB).ByID(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "post not found")
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func apiAdminPostsCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req apiPostRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		p := req.post()
		if !validateAPIPost(&p) {
			writeAPIError(w, http.StatusBadRequest, "title is required and status must be draft or published")
			return
		}
		ensurePublishedAt(&p)
		repo := db.NewPostRepo(d.DB)
		id, err := repo.Create(&p)
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
		created, err := repo.ByID(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func apiAdminPostsUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewPostRepo(d.DB)
		existing, err := repo.ByID(id)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, "post not found")
			return
		}
		var req apiPostRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		p := req.post()
		p.ID = id
		if p.Status == "" {
			p.Status = existing.Status
		}
		if !validateAPIPost(&p) {
			writeAPIError(w, http.StatusBadRequest, "title is required and status must be draft or published")
			return
		}
		if p.PublishedAt == nil {
			p.PublishedAt = existing.PublishedAt
		}
		ensurePublishedAt(&p)
		if err := repo.Update(&p); err != nil {
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
		updated, err := repo.ByID(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

func apiAdminPostsDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewPostRepo(d.DB)
		if _, err := repo.ByID(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "post not found")
			return
		}
		if err := repo.Delete(id); err != nil {
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

func apiAdminPostsStatusHandler(d Deps, status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := apiID(w, r)
		if !ok {
			return
		}
		repo := db.NewPostRepo(d.DB)
		if _, err := repo.ByID(id); err != nil {
			writeAPIError(w, http.StatusNotFound, "post not found")
			return
		}
		if err := repo.SetStatus(id, status); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "save failed")
			return
		}
		if err := d.Content.Reload(); err != nil {
			writeAPIError(w, http.StatusInternalServerError, "reload failed")
			return
		}
		updated, err := repo.ByID(id)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}
