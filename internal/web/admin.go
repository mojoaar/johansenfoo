package web

import "net/http"

func adminDashboardHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "dashboard", "Dashboard")
		renderAdmin(w, r, "admin_dashboard", data)
	}
}
