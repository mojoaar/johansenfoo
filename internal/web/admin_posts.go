package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminPostsGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		loc := siteLocation(c)
		posts, err := db.NewPostRepo(d.DB).All()
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		views := make([]postView, 0, len(posts))
		for _, p := range posts {
			views = append(views, toView(p, loc))
		}
		data := NewAdminPage(d, r, "posts", "Posts")
		data.Posts = views
		renderAdmin(w, r, "admin_posts", data)
	}
}

func adminPostNewHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "posts", "New post")
		data.IsNew = true
		renderAdmin(w, r, "admin_post_form", data)
	}
}

func adminPostEditHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		p, err := db.NewPostRepo(d.DB).ByID(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		c := d.Content.Current()
		data := NewAdminPage(d, r, "posts", "Edit post")
		data.Post = toView(*p, siteLocation(c))
		renderAdmin(w, r, "admin_post_form", data)
	}
}

func formPost(r *http.Request) *db.Post {
	title := r.FormValue("title")
	slug := db.Slugify(r.FormValue("slug"))
	if slug == "" {
		slug = db.Slugify(title)
	}
	p := &db.Post{
		Slug:           slug,
		Title:          title,
		Summary:        r.FormValue("summary"),
		BodyMD:         r.FormValue("body_md"),
		Status:         r.FormValue("status"),
		HeroImageURL:   r.FormValue("hero_image_url"),
		HeroImageAlt:   r.FormValue("hero_image_alt"),
		SEOTitle:       r.FormValue("seo_title"),
		SEODescription: r.FormValue("seo_description"),
		OGImageURL:     r.FormValue("og_image_url"),
		CanonicalURL:   r.FormValue("canonical_url"),
		NoIndex:        r.FormValue("noindex") != "",
	}
	for _, name := range strings.Split(r.FormValue("tags"), ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		p.Tags = append(p.Tags, db.Tag{Name: name, Slug: db.Slugify(name)})
	}
	return p
}

func ensurePublishedAt(p *db.Post) {
	if p.Status == "published" && p.PublishedAt == nil {
		now := time.Now().UTC()
		p.PublishedAt = &now
	}
}

func adminPostCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		p := formPost(r)
		ensurePublishedAt(p)
		if _, err := db.NewPostRepo(d.DB).Create(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/posts?saved=1", http.StatusSeeOther)
	}
}

func adminPostUpdateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		repo := db.NewPostRepo(d.DB)
		existing, err := repo.ByID(id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		p := formPost(r)
		p.ID = id
		if p.PublishedAt == nil {
			p.PublishedAt = existing.PublishedAt
		}
		ensurePublishedAt(p)
		if err := repo.Update(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/posts?saved=1", http.StatusSeeOther)
	}
}

func adminPostDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewPostRepo(d.DB).Delete(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/posts?saved=1", http.StatusSeeOther)
	}
}

func adminPostStatusHandler(d Deps, status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewPostRepo(d.DB).SetStatus(id, status); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/posts?saved=1", http.StatusSeeOther)
	}
}
