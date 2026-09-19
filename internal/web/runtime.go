package web

import (
	"bytes"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mojoaar/johansenfoo/internal/sysinfo"
)

type RuntimeStats struct {
	UptimeSeconds int64   `json:"uptime_seconds"`
	Goroutines    int     `json:"goroutines"`
	HeapAlloc     uint64  `json:"heap_alloc"`
	HeapSys       uint64  `json:"heap_sys"`
	GCCount       uint32  `json:"gc_count"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemUsed       uint64  `json:"mem_used"`
	MemLimit      uint64  `json:"mem_limit"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	Available     bool    `json:"container_stats_available"`
}

func collectRuntime(d Deps) RuntimeStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	dataDir := "."
	if d.Cfg != nil && d.Cfg.DBPath != "" {
		dataDir = filepath.Dir(d.Cfg.DBPath)
	}
	container, _ := sysinfo.Read(dataDir)

	return RuntimeStats{
		UptimeSeconds: int64(time.Since(d.Started).Seconds()),
		Goroutines:    runtime.NumGoroutine(),
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		GCCount:       m.NumGC,
		CPUPercent:    container.CPUPercent,
		MemUsed:       container.MemUsed,
		MemLimit:      container.MemLimit,
		DiskUsed:      container.DiskUsed,
		DiskTotal:     container.DiskTotal,
		Available:     container.Available,
	}
}

func metricsHandler() http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGoCollector())
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
}

func renderFragment(w http.ResponseWriter, name string, data page) {
	var buf bytes.Buffer
	if err := templates.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func adminRuntimeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := NewAdminPage(d, r, "dashboard", "Runtime")
		data.Runtime = collectRuntime(d)
		renderFragment(w, "admin_runtime", data)
	}
}
