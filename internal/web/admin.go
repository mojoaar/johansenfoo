package web

import "net/http"

func adminDashboardHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			http.Error(w, "content unavailable", http.StatusInternalServerError)
			return
		}
		data := NewAdminPage(d, r, "dashboard", "Dashboard")
		data.Profile = c.Profile
		data.Social = c.Social
		data.Projects = c.Projects
		data.Experience = c.Experience
		data.Skills = c.Skills
		data.ThemeSlug = c.Theme.Slug
		renderAdmin(w, r, "admin_dashboard", data)
	}
}
