package web

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func adminProfileGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "profile", "Profile")
		data.Profile = c.Profile
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_profile", data)
	}
}

func adminProfilePostHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		p := &db.Profile{
			Name:       r.FormValue("name"),
			Handle:     r.FormValue("handle"),
			Location:   r.FormValue("location"),
			DOB:        r.FormValue("dob"),
			Tagline:    r.FormValue("tagline"),
			HeroBio:    r.FormValue("hero_bio"),
			Bio:        r.FormValue("bio"),
			AboutPara1: r.FormValue("about_para_1"),
			AboutPara2: r.FormValue("about_para_2"),
			Avatar:     r.FormValue("avatar"),
		}
		if err := db.NewProfileRepo(d.DB).Update(p); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/profile?saved=1", http.StatusSeeOther)
	}
}

func adminSocialGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "social", "Social links")
		data.Social = c.Social
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_social", data)
	}
}

func adminSocialCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		l := &db.SocialLink{
			Platform: r.FormValue("platform"),
			URL:      r.FormValue("url"),
			Label:    r.FormValue("label"),
			Sort:     formInt(r, "sort"),
			Visible:  r.FormValue("visible") != "",
		}
		if _, err := db.NewProfileRepo(d.DB).CreateSocialLink(l); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func adminSocialUpdateHandler(d Deps) http.HandlerFunc {
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
		l := &db.SocialLink{
			ID:       id,
			Platform: r.FormValue("platform"),
			URL:      r.FormValue("url"),
			Label:    r.FormValue("label"),
			Sort:     formInt(r, "sort"),
			Visible:  r.FormValue("visible") != "",
		}
		if err := db.NewProfileRepo(d.DB).UpdateSocialLink(l); err != nil {
			http.Error(w, "save failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func adminSocialDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "bad id", http.StatusBadRequest)
			return
		}
		if err := db.NewProfileRepo(d.DB).DeleteSocialLink(id); err != nil {
			http.Error(w, "delete failed", http.StatusInternalServerError)
			return
		}
		if err := d.Content.Reload(); err != nil {
			http.Error(w, "reload failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/social?saved=1", http.StatusSeeOther)
	}
}

func formInt(r *http.Request, key string) int {
	v, err := strconv.Atoi(r.FormValue(key))
	if err != nil {
		return 0
	}
	return v
}
