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

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
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
		ar.Get("/social", adminSocialGetHandler(d))
		ar.Post("/social", adminSocialCreateHandler(d))
		ar.Post("/social/{id}", adminSocialUpdateHandler(d))
		ar.Post("/social/{id}/delete", adminSocialDeleteHandler(d))
	})

	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
