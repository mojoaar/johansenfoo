package web

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/mojoaar/johansenfoo/internal/db"
)

const bearerPrefix = "Bearer "

func apiAuthMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if currentSession(d, r) {
				next.ServeHTTP(w, r)
				return
			}
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				writeAPIError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			presented := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
			key, err := db.NewSettingsRepo(d.DB).Get(apiKeySettingKey)
			if err != nil || key == "" || presented == "" ||
				subtle.ConstantTimeCompare([]byte(presented), []byte(key)) != 1 {
				writeAPIError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
