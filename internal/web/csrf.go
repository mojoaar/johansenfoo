package web

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func ensureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	token := randomToken()
	if w != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     csrfCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			Secure:   secureRequest(r),
		})
	}
	return token
}

func csrfExempt(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if strings.EqualFold(r.Header.Get("HX-Request"), "true") {
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	switch r.URL.Path {
	case "/login", "/setup":
		return true
	}
	return r.URL.Path == "/mcp" || strings.HasPrefix(r.URL.Path, "/mcp/")
}

func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if csrfExempt(r) {
			next.ServeHTTP(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		cookie, err := r.Cookie(csrfCookieName)
		if err != nil || cookie.Value == "" {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		if subtle.ConstantTimeCompare([]byte(r.PostFormValue("csrf_token")), []byte(cookie.Value)) != 1 {
			http.Error(w, "invalid csrf token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func methodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err == nil {
				switch m := strings.ToUpper(r.Form.Get("_method")); m {
				case http.MethodPut, http.MethodPatch, http.MethodDelete:
					r.Method = m
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
