package web

import (
	"encoding/json"
	"net/http"
)

type meProject struct {
	Name        string  `json:"name"`
	URL         *string `json:"url"`
	Description string  `json:"description"`
}

type meResponse struct {
	Name     string            `json:"name"`
	Handle   string            `json:"handle"`
	Location string            `json:"location"`
	DOB      string            `json:"dob"`
	Bio      string            `json:"bio"`
	Skills   []string          `json:"skills"`
	Social   map[string]string `json:"social"`
	Projects []meProject       `json:"projects"`
}

func meHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content

		skills := make([]string, 0, len(c.Skills))
		for _, s := range c.Skills {
			skills = append(skills, s.Name)
		}

		projects := make([]meProject, 0, len(c.Projects))
		for _, p := range c.Projects {
			var url *string
			if p.URL != "" {
				u := p.URL
				url = &u
			}
			projects = append(projects, meProject{
				Name:        p.Name,
				URL:         url,
				Description: p.Description,
			})
		}

		social := make(map[string]string, len(c.Social))
		for _, s := range c.Social {
			social[s.Platform] = s.URL
		}

		payload, err := json.MarshalIndent(meResponse{
			Name:     c.Profile.Name,
			Handle:   c.Profile.Handle,
			Location: c.Profile.Location,
			DOB:      c.Profile.DOB,
			Bio:      c.Profile.Bio,
			Skills:   skills,
			Social:   social,
			Projects: projects,
		}, "", "  ")
		if err != nil {
			http.Error(w, "encode error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(append(payload, '\n'))
	}
}
