package web

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "johansenfoo_session"
	csrfCookieName    = "johansenfoo_csrf"
	passwordHashKey   = "admin_password_hash"
	sessionLifetime   = 30 * 24 * time.Hour
	minPasswordLength = 12
	loginMaxAttempts  = 10
	loginWindow       = 15 * time.Minute
)

func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func secureRequest(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

func passwordHash(d Deps) (string, bool, error) {
	v, err := db.NewSettingsRepo(d.DB).Get(passwordHashKey)
	if errors.Is(err, db.ErrSettingNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, v != "", nil
}

func startSession(d Deps, w http.ResponseWriter, r *http.Request) error {
	id := randomToken()
	expires := time.Now().UTC().Add(sessionLifetime)
	if err := db.NewSessionRepo(d.DB).Create(id, expires); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secureRequest(r),
		Expires:  expires,
	})
	return nil
}

func currentSession(d Deps, r *http.Request) bool {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return false
	}
	s, err := db.NewSessionRepo(d.DB).Get(c.Value)
	if err != nil {
		return false
	}
	return s.ExpiresAt.After(time.Now().UTC())
}

func pruneSessions(d *sql.DB) (int64, error) {
	return db.NewSessionRepo(d).DeleteExpired()
}

func authMiddleware(d Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !currentSession(d, r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func setupHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if configured {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			renderAdmin(w, r, "admin_setup", data)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		pw := r.FormValue("password")
		if len(pw) < minPasswordLength {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			data.Error = "Password must be at least 12 characters."
			renderAdmin(w, r, "admin_setup", data, http.StatusBadRequest)
			return
		}
		if confirm := r.FormValue("password_confirm"); confirm != "" && pw != confirm {
			data := NewAdminPage(d, r, "setup", "Set up admin access")
			data.Error = "Passwords do not match."
			renderAdmin(w, r, "admin_setup", data, http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "hash error", http.StatusInternalServerError)
			return
		}
		if err := db.NewSettingsRepo(d.DB).Set(passwordHashKey, string(hash)); err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if err := startSession(d, w, r); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func loginHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hash, configured, err := passwordHash(d)
		if err != nil {
			http.Error(w, "storage error", http.StatusInternalServerError)
			return
		}
		if !configured {
			http.Redirect(w, r, "/setup", http.StatusSeeOther)
			return
		}
		if r.Method == http.MethodGet {
			data := NewAdminPage(d, r, "login", "Sign in")
			renderAdmin(w, r, "admin_login", data)
			return
		}
		if !loginLimiter.allow(clientIP(r), loginMaxAttempts, loginWindow) {
			http.Error(w, "too many attempts", http.StatusTooManyRequests)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.FormValue("password"))) != nil {
			data := NewAdminPage(d, r, "login", "Sign in")
			data.Error = "Incorrect password."
			renderAdmin(w, r, "admin_login", data, http.StatusUnauthorized)
			return
		}
		if err := startSession(d, w, r); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func logoutHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
			_ = db.NewSessionRepo(d.DB).Delete(c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   secureRequest(r),
			MaxAge:   -1,
		})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
}

type rateEntry struct {
	count   int
	resetAt time.Time
}

func newRateLimiter() *rateLimiter {
	rl := &rateLimiter{entries: make(map[string]*rateEntry)}
	go func() {
		for range time.Tick(time.Minute) {
			rl.sweep()
		}
	}()
	return rl
}

func (rl *rateLimiter) sweep() {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for k, e := range rl.entries {
		if now.After(e.resetAt) {
			delete(rl.entries, k)
		}
	}
}

func (rl *rateLimiter) allow(key string, max int, window time.Duration) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.entries[key]
	if !ok || now.After(e.resetAt) {
		rl.entries[key] = &rateEntry{count: 1, resetAt: now.Add(window)}
		return true
	}
	if e.count >= max {
		return false
	}
	e.count++
	return true
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

var loginLimiter = newRateLimiter()
