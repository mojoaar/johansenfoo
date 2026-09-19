package web

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mojoaar/johansenfoo/internal/db"
)

func hashIP(salt, ip string) string {
	sum := sha256.Sum256([]byte(salt + ":" + ip))
	return hex.EncodeToString(sum[:])
}

type viewRecorder struct {
	db   *sql.DB
	mu   sync.Mutex
	salt string
	day  string
}

func newViewRecorder(d *sql.DB) *viewRecorder {
	return &viewRecorder{db: d}
}

func (v *viewRecorder) saltFor(day string) (string, error) {
	repo := db.NewSettingsRepo(v.db)
	storedDay, err := repo.Get("stats_salt_date")
	if err == nil && storedDay == day {
		if salt, err := repo.Get("stats_salt"); err == nil && salt != "" {
			return salt, nil
		}
	}
	salt := randomToken()
	if err := repo.Set("stats_salt", salt); err != nil {
		return "", err
	}
	if err := repo.Set("stats_salt_date", day); err != nil {
		return "", err
	}
	return salt, nil
}

func (v *viewRecorder) hash(ip string) (string, error) {
	day := time.Now().UTC().Format("2006-01-02")
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.salt == "" || v.day != day {
		salt, err := v.saltFor(day)
		if err != nil {
			return "", err
		}
		v.salt = salt
		v.day = day
	}
	return hashIP(v.salt, ip), nil
}

func skipStatsPath(path string) bool {
	switch path {
	case "/login", "/setup", "/logout":
		return true
	}
	for _, prefix := range []string{"/admin", "/static", "/api", "/mcp", "/metrics"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

type viewResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *viewResponseWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *viewResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func newPageViewMiddleware(d Deps) func(http.Handler) http.Handler {
	rec := newViewRecorder(d.DB)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			vw := &viewResponseWriter{ResponseWriter: w}
			next.ServeHTTP(vw, r)

			if r.Method != http.MethodGet || skipStatsPath(r.URL.Path) {
				return
			}
			if vw.status < 200 || vw.status >= 300 {
				return
			}
			if !strings.Contains(vw.Header().Get("Content-Type"), "text/html") {
				return
			}
			if c := d.Content.Current(); c != nil && c.Settings["stats_enabled"] == "false" {
				return
			}
			hash, err := rec.hash(clientIP(r))
			if err != nil {
				return
			}
			_ = db.NewPageViewRepo(d.DB).Record(&db.PageView{
				Path:      r.URL.Path,
				Referrer:  r.Referer(),
				UserAgent: r.UserAgent(),
				IPHash:    hash,
			})
		})
	}
}

func pruneViews(d *sql.DB) (int64, error) {
	days := 90
	if v, err := db.NewSettingsRepo(d).Get("stats_retention_days"); err == nil {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	return db.NewPageViewRepo(d).Prune(cutoff)
}

func startStatsPruner(d *sql.DB, interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			_, _ = pruneViews(d)
		}
	}()
	return ticker.Stop
}
