package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/markdown"
)

type apiPost struct {
	db.Post
	PublishedAt *string `json:"published_at"`
	HTML        string  `json:"html"`
}

type apiPostsResponse struct {
	Posts   []apiPost `json:"posts"`
	Page    int       `json:"page"`
	PerPage int       `json:"per_page"`
	Total   int       `json:"total"`
}

func toAPIPost(p db.Post, loc *time.Location) apiPost {
	ap := apiPost{Post: p, HTML: string(markdown.Render(p.BodyMD))}
	if p.PublishedAt != nil {
		s := p.PublishedAt.In(loc).Format(time.RFC3339)
		ap.PublishedAt = &s
	}
	return ap
}

func apiPostsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		if !postsEnabled(c) {
			writeAPIError(w, http.StatusNotFound, "posts are disabled")
			return
		}
		repo := db.NewPostRepo(d.DB)
		pageNum := pageParam(r)
		offset := (pageNum - 1) * postsPerPage
		tag := r.URL.Query().Get("tag")

		var (
			posts []db.Post
			total int
			err   error
		)
		if tag != "" {
			total, err = repo.CountByTag(tag)
			if err == nil {
				posts, err = repo.ByTag(tag, postsPerPage, offset)
			}
		} else {
			total, err = repo.CountPublished()
			if err == nil {
				posts, err = repo.Published(postsPerPage, offset)
			}
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		out := make([]apiPost, 0, len(posts))
		loc := siteLocation(c)
		for _, p := range posts {
			out = append(out, toAPIPost(p, loc))
		}
		writeJSON(w, http.StatusOK, apiPostsResponse{
			Posts:   out,
			Page:    pageNum,
			PerPage: postsPerPage,
			Total:   total,
		})
	}
}

func apiPostHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		if !postsEnabled(c) {
			writeAPIError(w, http.StatusNotFound, "posts are disabled")
			return
		}
		p, err := db.NewPostRepo(d.DB).PublishedBySlug(chi.URLParam(r, "slug"))
		if errors.Is(err, db.ErrPostNotFound) {
			writeAPIError(w, http.StatusNotFound, "post not found")
			return
		}
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "storage error")
			return
		}
		writeJSON(w, http.StatusOK, toAPIPost(*p, siteLocation(c)))
	}
}
