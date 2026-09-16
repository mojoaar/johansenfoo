package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunBootsAndServes(t *testing.T) {
	h, _, cleanup, err := buildHandler(t.TempDir())
	if err != nil {
		t.Fatalf("buildHandler: %v", err)
	}
	defer cleanup()

	cases := []struct {
		path string
		want int
	}{
		{"/", http.StatusOK},
		{"/me", http.StatusOK},
		{"/health", http.StatusOK},
		{"/robots.txt", http.StatusOK},
		{"/sitemap.xml", http.StatusOK},
		{"/static/style.css", http.StatusOK},
		{"/nope", http.StatusNotFound},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("GET %s = %d, want %d", c.path, rec.Code, c.want)
		}
	}
}
