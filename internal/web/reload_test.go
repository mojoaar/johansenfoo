package web

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mojoaar/johansenfoo/internal/config"
	"github.com/mojoaar/johansenfoo/internal/db"
)

func TestHandlersServeUpdatedContentAfterReload(t *testing.T) {
	cases := []struct {
		name   string
		path   string
		mutate func(*sql.DB) error
		want   string
		stale  string
	}{
		{
			name: "/me",
			path: "/me",
			mutate: func(d *sql.DB) error {
				_, err := d.Exec(`UPDATE profile SET name = ? WHERE id = 1`, "Reloaded Name")
				return err
			},
			want:  `"name": "Reloaded Name"`,
			stale: `"name": "Morten Johansen"`,
		},
		{
			name: "/robots.txt",
			path: "/robots.txt",
			mutate: func(d *sql.DB) error {
				return db.NewSettingsRepo(d).Set("robots_txt", "User-agent: ReloadBot\nDisallow: /\n")
			},
			want:  "User-agent: ReloadBot",
			stale: "johansen.foo/sitemap.xml",
		},
		{
			name: "/sitemap.xml",
			path: "/sitemap.xml",
			mutate: func(d *sql.DB) error {
				return db.NewSettingsRepo(d).Set("canonical_base_url", "https://reloaded.example")
			},
			want:  "<loc>https://reloaded.example/</loc>",
			stale: "<loc>https://johansen.foo/</loc>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := newTestDB(t)
			store, err := NewContentStore(d)
			if err != nil {
				t.Fatalf("NewContentStore: %v", err)
			}
			h := New(Deps{
				DB:      d,
				Cfg:     &config.Config{Port: 8080, BaseURL: "https://johansen.foo"},
				Content: store,
				Version: "test",
				Started: time.Now(),
			})

			if err := tc.mutate(d); err != nil {
				t.Fatalf("mutate: %v", err)
			}
			if err := store.Reload(); err != nil {
				t.Fatalf("Reload: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, tc.want) {
				t.Errorf("response for %s did not serve the updated value %q\ngot:\n%s", tc.path, tc.want, body)
			}
			if strings.Contains(body, tc.stale) {
				t.Errorf("response for %s still served stale value %q after Reload", tc.path, tc.stale)
			}
		})
	}
}
