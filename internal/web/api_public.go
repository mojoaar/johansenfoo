package web

import (
	"net/http"

	"github.com/mojoaar/johansenfoo/internal/db"
	"github.com/mojoaar/johansenfoo/internal/theme"
)

type apiProfileResponse struct {
	db.Profile
	Social []db.SocialLink `json:"social"`
}

type apiThemeResponse struct {
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Light       map[string]string `json:"light"`
	Dark        map[string]string `json:"dark"`
}

func apiProfileHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, apiProfileResponse{Profile: c.Profile, Social: visibleSocial(c.Social)})
	}
}

func apiProjectsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleProjects(c.Projects))
	}
}

func apiExperienceHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleExperience(c.Experience))
	}
}

func apiSkillsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		writeJSON(w, http.StatusOK, visibleSkills(c.Skills))
	}
}

func apiThemeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := d.Content.Current()
		if c == nil {
			writeAPIError(w, http.StatusInternalServerError, "content unavailable")
			return
		}
		t := c.Theme
		writeJSON(w, http.StatusOK, apiThemeResponse{
			Slug:        t.Slug,
			Name:        t.Name,
			Description: t.Description,
			Light:       theme.Resolve(t.TokensBase, t.TokensLight),
			Dark:        theme.Resolve(t.TokensBase, t.TokensDark),
		})
	}
}
