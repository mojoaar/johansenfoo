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
		data.StatsEnabled = c.Settings["stats_enabled"] != "false"
		data.Settings = c.Settings
		data.ViewsToday = countSinceDays(d, 1)
		data.Views7d = countSinceDays(d, 7)
		data.Views30d = countSinceDays(d, 30)
		if data.StatsEnabled {
			if stats, err := collectVisitors(d, "7d"); err == nil {
				data.DailyUniques = stats.DailyUniques
				data.TopPaths = stats.TopPaths
				data.TopReferrers = stats.TopReferrers
				data.RecentHits = stats.Recent
			}
		}
		renderAdmin(w, r, "admin_dashboard", data)
	}
}
