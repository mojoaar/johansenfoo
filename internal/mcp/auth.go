package mcp

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	rateLimitMax    = 100
	rateLimitWindow = time.Minute
)

const bearerPrefix = "Bearer "

type limiter struct {
	mu      sync.Mutex
	entries map[string]*limitEntry
	max     int
	window  time.Duration
}

type limitEntry struct {
	count   int
	resetAt time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{entries: make(map[string]*limitEntry), max: max, window: window}
}

func (l *limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[key]
	if !ok || now.After(e.resetAt) {
		l.entries[key] = &limitEntry{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if e.count >= l.max {
		return false
	}
	e.count++
	return true
}

func (l *limiter) sweep() {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, e := range l.entries {
		if now.After(e.resetAt) {
			delete(l.entries, key)
		}
	}
}

func bearerAuthorized(r *http.Request, key string) bool {
	if key == "" {
		return false
	}
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return false
	}
	presented := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	if presented == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(presented), []byte(key)) == 1
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
