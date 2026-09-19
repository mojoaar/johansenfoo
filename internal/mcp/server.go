package mcp

import (
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func NewServer(b Backend, version string) *server.MCPServer {
	if version == "" {
		version = "dev"
	}
	s := server.NewMCPServer("johansenfoo", version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	return s
}

func Handler(d Deps) http.Handler {
	stream := server.NewStreamableHTTPServer(NewServer(Backend{DB: d.DB, Reload: d.Reload}, d.Version))
	limiter := newLimiter(rateLimitMax, rateLimitWindow)
	go func() {
		for range time.Tick(time.Minute) {
			limiter.sweep()
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := d.APIKey()
		if err != nil || !bearerAuthorized(r, key) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !limiter.allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		stream.ServeHTTP(w, r)
	})
}
