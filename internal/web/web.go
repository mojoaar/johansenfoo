package web

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mojoaar/johansenfoo/internal/config"
)

type Deps struct {
	DB      *sql.DB
	Cfg     *config.Config
	Content *ContentStore
	Version string
	Started time.Time
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(securityHeaders)
	r.Use(methodOverride)
	r.Use(csrfMiddleware)
	r.Use(middleware.Timeout(30 * time.Second))

	static, err := staticHandler()
	if err != nil {
		panic(err)
	}
	r.Handle("/static/*", static)

	r.Get("/", landingHandler(d))
	r.Get("/me", meHandler(d))
	r.Get("/robots.txt", robotsHandler(d))
	r.Get("/sitemap.xml", sitemapHandler(d))
	r.Get("/health", healthHandler(d))

	r.Get("/posts", postIndexHandler(d))
	r.Get("/posts/{slug}", postHandler(d))
	r.Get("/tags/{slug}", tagArchiveHandler(d))
	r.Get("/feed.xml", feedHandler(d))

	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/profile", apiProfileHandler(d))
		api.Get("/projects", apiProjectsHandler(d))
		api.Get("/experience", apiExperienceHandler(d))
		api.Get("/skills", apiSkillsHandler(d))
		api.Get("/theme", apiThemeHandler(d))
		api.Get("/posts", apiPostsHandler(d))
		api.Get("/posts/{slug}", apiPostHandler(d))

		api.Route("/admin", func(apiAdmin chi.Router) {
			apiAdmin.Use(apiAuthMiddleware(d))
			apiAdmin.Get("/profile", apiAdminProfileGetHandler(d))
			apiAdmin.Put("/profile", apiAdminProfilePutHandler(d))
			apiAdmin.Get("/social", apiAdminSocialListHandler(d))
			apiAdmin.Post("/social", apiAdminSocialCreateHandler(d))
			apiAdmin.Put("/social/{id}", apiAdminSocialUpdateHandler(d))
			apiAdmin.Delete("/social/{id}", apiAdminSocialDeleteHandler(d))
			apiAdmin.Route("/projects", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Post("/", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Get("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Put("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
				r.Delete("/{id}", apiCRUDHandler(d, projectCRUD(d), "project"))
			})
			apiAdmin.Route("/experience", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Post("/", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Get("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Put("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
				r.Delete("/{id}", apiCRUDHandler(d, experienceCRUD(d), "experience"))
			})
			apiAdmin.Route("/skills", func(r chi.Router) {
				r.Get("/", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Post("/", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Get("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Put("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
				r.Delete("/{id}", apiCRUDHandler(d, skillCRUD(d), "skill"))
			})
			apiAdmin.Route("/posts", func(r chi.Router) {
				r.Get("/", apiAdminPostsListHandler(d))
				r.Post("/", apiAdminPostsCreateHandler(d))
				r.Get("/{id}", apiAdminPostsGetHandler(d))
				r.Put("/{id}", apiAdminPostsUpdateHandler(d))
				r.Delete("/{id}", apiAdminPostsDeleteHandler(d))
				r.Post("/{id}/publish", apiAdminPostsStatusHandler(d, "published"))
				r.Post("/{id}/unpublish", apiAdminPostsStatusHandler(d, "draft"))
			})
			apiAdmin.Put("/settings/posts", apiAdminPostsSettingHandler(d))
			apiAdmin.Put("/settings/theme", apiAdminThemeSettingHandler(d))
			apiAdmin.Get("/export", apiAdminExportHandler(d))
			apiAdmin.Post("/import", apiAdminImportHandler(d))
		})
	})

	r.Get("/setup", setupHandler(d))
	r.Post("/setup", setupHandler(d))
	r.Get("/login", loginHandler(d))
	r.Post("/login", loginHandler(d))
	r.Post("/logout", logoutHandler(d))

	r.Route("/admin", func(ar chi.Router) {
		ar.Use(authMiddleware(d))
		ar.Get("/", adminDashboardHandler(d))
		ar.Get("/profile", adminProfileGetHandler(d))
		ar.Post("/profile", adminProfilePostHandler(d))
		ar.Put("/profile", adminProfilePostHandler(d))
		ar.Get("/social", adminSocialGetHandler(d))
		ar.Post("/social", adminSocialCreateHandler(d))
		ar.Post("/social/{id}", adminSocialUpdateHandler(d))
		ar.Put("/social/{id}", adminSocialUpdateHandler(d))
		ar.Post("/social/{id}/delete", adminSocialDeleteHandler(d))
		ar.Delete("/social/{id}/delete", adminSocialDeleteHandler(d))
		ar.Get("/projects", adminProjectsGetHandler(d))
		ar.Get("/projects/new", adminProjectNewHandler(d))
		ar.Post("/projects", adminProjectCreateHandler(d))
		ar.Get("/projects/{id}", adminProjectEditHandler(d))
		ar.Post("/projects/{id}", adminProjectUpdateHandler(d))
		ar.Put("/projects/{id}", adminProjectUpdateHandler(d))
		ar.Post("/projects/{id}/delete", adminProjectDeleteHandler(d))
		ar.Delete("/projects/{id}/delete", adminProjectDeleteHandler(d))
		ar.Get("/experience", adminExperienceGetHandler(d))
		ar.Get("/experience/new", adminExperienceNewHandler(d))
		ar.Post("/experience", adminExperienceCreateHandler(d))
		ar.Get("/experience/{id}", adminExperienceEditHandler(d))
		ar.Post("/experience/{id}", adminExperienceUpdateHandler(d))
		ar.Put("/experience/{id}", adminExperienceUpdateHandler(d))
		ar.Post("/experience/{id}/delete", adminExperienceDeleteHandler(d))
		ar.Delete("/experience/{id}/delete", adminExperienceDeleteHandler(d))
		ar.Get("/skills", adminSkillsGetHandler(d))
		ar.Post("/skills", adminSkillCreateHandler(d))
		ar.Post("/skills/{id}", adminSkillUpdateHandler(d))
		ar.Put("/skills/{id}", adminSkillUpdateHandler(d))
		ar.Post("/skills/{id}/delete", adminSkillDeleteHandler(d))
		ar.Delete("/skills/{id}/delete", adminSkillDeleteHandler(d))
		ar.Get("/posts", adminPostsGetHandler(d))
		ar.Get("/posts/new", adminPostNewHandler(d))
		ar.Post("/posts", adminPostCreateHandler(d))
		ar.Get("/posts/{id}", adminPostEditHandler(d))
		ar.Post("/posts/{id}", adminPostUpdateHandler(d))
		ar.Put("/posts/{id}", adminPostUpdateHandler(d))
		ar.Post("/posts/{id}/delete", adminPostDeleteHandler(d))
		ar.Delete("/posts/{id}/delete", adminPostDeleteHandler(d))
		ar.Post("/posts/{id}/publish", adminPostStatusHandler(d, "published"))
		ar.Put("/posts/{id}/publish", adminPostStatusHandler(d, "published"))
		ar.Post("/posts/{id}/unpublish", adminPostStatusHandler(d, "draft"))
		ar.Put("/posts/{id}/unpublish", adminPostStatusHandler(d, "draft"))
		ar.Get("/security", adminSecurityGetHandler(d))
		ar.Post("/security/password", adminPasswordChangeHandler(d))
		ar.Put("/security/password", adminPasswordChangeHandler(d))
		ar.Post("/security/apikey", adminAPIKeyRegenerateHandler(d))
		ar.Put("/security/apikey", adminAPIKeyRegenerateHandler(d))
	})

	startSessionPruner(d.DB, time.Hour)

	return r
}

func startSessionPruner(d *sql.DB, interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			_, _ = pruneSessions(d)
		}
	}()
	return ticker.Stop
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
