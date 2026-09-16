package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const apiKeySettingKey = "api_key"

func adminSecurityGetHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		data := NewAdminPage(d, r, "security", "Security")
		data.ThemeSlug = c.Theme.Slug
		if r.URL.Query().Get("key") == "1" {
			if key, err := db.NewSettingsRepo(d.DB).Get(apiKeySettingKey); err == nil {
				data.APIKey = key
			}
		}
		renderAdmin(w, r, "admin_security", data)
	}
}

func adminPasswordChangeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		current := r.FormValue("current_password")
		next := r.FormValue("new_password")

		hash, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if !configured || bcrypt.CompareHashAndPassword([]byte(hash), []byte(current)) != nil {
			http.Error(w, "current password is incorrect", http.StatusBadRequest)
			return
		}
		if len(next) < minPasswordLength {
			http.Error(w, "new password is too short", http.StatusBadRequest)
			return
		}
		if next != r.FormValue("confirm_password") {
			http.Error(w, "new passwords do not match", http.StatusBadRequest)
			return
		}
		newHash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hash error", http.StatusInternalServerError)
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set(passwordHashKey, string(newHash)); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/security?saved=1", http.StatusSeeOther)
	}
}

func adminAPIKeyRegenerateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := randomToken() + randomToken()
		if err := db.NewSettingsRepo(d.DB).Set(apiKeySettingKey, key); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/security?key=1", http.StatusSeeOther)
	}
}
