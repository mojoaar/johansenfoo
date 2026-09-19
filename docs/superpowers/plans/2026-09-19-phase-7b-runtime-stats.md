# Phase 7b — Runtime Stats Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Expose runtime and container statistics — Go process figures and container CPU/memory/disk — at `/metrics`, on the admin dashboard (polled over HTMX), over `GET /api/v1/admin/stats/system`, and through the `get_system_stats` MCP tool.

**Architecture:** A new `internal/sysinfo` package reads container CPU/memory from cgroup v2 and disk from `syscall.Statfs`, with a `!linux` fallback that reports "unavailable". `internal/web` builds a `RuntimeStats` snapshot from `runtime.MemStats`, the server start time and sysinfo, serves Prometheus at `/metrics`, renders an HTMX-polled dashboard partial at `/admin/runtime`, and exposes the REST endpoint. `internal/mcp` gains `get_system_stats`.

**Tech Stack:** Go 1.25, `github.com/prometheus/client_golang`, stdlib `runtime`, `syscall`, `os`.

**Spec:** `docs/superpowers/specs/2026-09-16-johansenfoo-agent-first-site-design.md` (Runtime Stats; admin dashboard runtime cards; MCP stats tools)

## Global Constraints

- `CGO_ENABLED=0`; module `github.com/mojoaar/johansenfoo`, Go 1.25.5. No comments unless essential.
- No comments except `//go:build` (required) and `//go:embed`.
- Migrations append-only; this phase adds none.
- `/metrics` is excluded from page-view recording (already in `skipStatsPath`).
- Must degrade to "unavailable" off Linux rather than crash.
- Every task ends with build/test/vet/gofmt clean, then a commit.
- Docs are part of the definition of done.

## Scope

In scope: `internal/sysinfo`, Prometheus `/metrics`, the runtime snapshot, `GET /api/v1/admin/stats/system`, `get_system_stats`, and the dashboard runtime cards with HTMX polling.

Out of scope: visit statistics (7a, merged); any new migration.

## Review Focus

1. **Degradation** — off Linux, sysinfo reports unavailable, never errors or panics.
2. **/metrics** — serves Prometheus text; is never page-view-recorded; requires no auth (operational endpoint).
3. **HTMX partial** — `/admin/runtime` returns a fragment (no full layout) behind auth; the dashboard polls it every ~5s.
4. **Accuracy** — CPU percent is a real delta over a sample window, not a cumulative counter mislabelled as a percent; absent values are reported as unavailable rather than zero-as-fact.
5. **No regression** — adding `/metrics` and the runtime endpoint does not disturb existing routes or the MCP transport.

---

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/sysinfo/sysinfo.go` | `Stats`, parsers, `Read` dispatch |
| `internal/sysinfo/sysinfo_linux.go` | cgroup v2 + `/proc` + `Statfs` reader |
| `internal/sysinfo/sysinfo_other.go` | `!linux` unavailable fallback |
| `internal/sysinfo/sysinfo_test.go` | Parser + fallback tests |
| `internal/web/runtime.go` | `RuntimeStats`, `collectRuntime`, `/metrics` handler, `/admin/runtime` partial |
| `internal/web/runtime_test.go` | Snapshot/partial/metrics tests |
| `internal/web/templates/admin_runtime.html` | Partial fragment |
| `internal/web/templates/admin_dashboard.html` | Polling container |
| `internal/web/api_admin_stats.go` | `GET /api/v1/admin/stats/system` |
| `internal/mcp/tools_stats.go` | `get_system_stats` |
| `README.md`, `AGENTS.md`, `CHANGELOG.md` | Docs |

---

### Task 1: `internal/sysinfo`

**Files:** create the four files above.

**Interfaces:**
- `sysinfo.Stats{Available bool; CPUPercent float64; MemUsed, MemLimit, DiskUsed, DiskTotal uint64}`
- `sysinfo.Read(dataDir string) (Stats, error)`
- Parsers used by the Linux reader and unit-tested everywhere: `parseCPUmax(s string) (quota, period int64, unlimited bool)`, `cpuPercent(deltaUsec int64, wall time.Duration) float64`.

- [ ] **Step 1: Failing tests** — `parseCPUmax("200000 100000")` → 200000/100000/false; `parseCPUmax("max 100000")` → unlimited; `cpuPercent(500000, time.Second)` → 50; `Read(t.TempDir())` on this host returns no error and, when `Available` is false, zero counters.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement.** Linux: read `cpu.stat` usage twice ~200ms apart for `CPUPercent`, `memory.current`/`memory.max` for memory, `Statfs(dataDir)` for disk. `!linux`: `Stats{Available: false}`.
- [ ] **Step 4: Pass + gate + commit** `feat: add container runtime stats`.

---

### Task 2: Prometheus `/metrics` and the runtime snapshot

**Files:** create `internal/web/runtime.go`, `internal/web/runtime_test.go`; modify `internal/web/web.go`; add the dependency.

**Interfaces:**
- `RuntimeStats{UptimeSeconds int64; Goroutines int; HeapAlloc, HeapSys uint64; GCCount uint32; CPUPercent float64; MemUsed, MemLimit, DiskUsed, DiskTotal uint64; Available bool}`
- `collectRuntime(d Deps) RuntimeStats` (uses `d.Started`, `runtime.ReadMemStats`, `sysinfo.Read(dataDir)`)
- `metricsHandler() http.Handler` with a private Prometheus registry (Go + process collectors)

- [ ] **Step 1: Failing tests** — a `GET /metrics` returns 200 `text/plain` with `go_goroutines`; the runtime snapshot has `Goroutines > 0` and non-negative uptime; `GET /metrics` is not page-view-recorded.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement + mount** `r.Handle("/metrics", metricsHandler())`.
- [ ] **Step 4: Pass + gate + commit** `feat: expose /metrics and a runtime snapshot`.

---

### Task 3: REST and MCP system stats

**Files:** modify `internal/web/api_admin_stats.go`, `internal/web/web.go`, `internal/mcp/tools_stats.go`, `internal/mcp/server.go`, `internal/mcp/backend.go`, tests.

**Interfaces:** `GET /api/v1/admin/stats/system` returns `RuntimeStats`; MCP `get_system_stats` returns the same. `mcp.Backend`/`Deps` gain `Started time.Time` and `DataDir string`.

- [ ] **Step 1: Failing tests** — REST returns 200 with `goroutines > 0` and 401 unauthenticated; MCP `get_system_stats` returns goroutines and an uptime.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement**, wiring `Started`/`DataDir` from `web.New` into the MCP deps.
- [ ] **Step 4: Pass + gate + commit** `feat: add system stats to REST and MCP`.

---

### Task 4: Dashboard runtime cards, docs and end-to-end

**Files:** create `internal/web/templates/admin_runtime.html`; modify `internal/web/runtime.go`, `internal/web/web.go`, `internal/web/templates/admin_dashboard.html`, `README.md`, `AGENTS.md`, `CHANGELOG.md`.

**Interfaces:** `adminRuntimeHandler(d)` renders the `admin_runtime` partial (no layout). The dashboard embeds `<div hx-get="/admin/runtime" hx-trigger="load, every 5s" hx-swap="innerHTML">`.

- [ ] **Step 1: Failing tests** — `GET /admin/runtime` (session) returns a fragment containing "Goroutines" and no `<html`; unauthenticated redirects/401; the dashboard contains `hx-get="/admin/runtime"`.
- [ ] **Step 2: Run to fail.**
- [ ] **Step 3: Implement** the partial + route + dashboard container; docs (README runtime section, AGENTS tree/routes/tools and the degrade-to-unavailable invariant, CHANGELOG).
- [ ] **Step 4: Manual E2E** — run against `/tmp/jf-7b`; `GET /metrics` shows `go_goroutines`; `/admin/runtime` renders cards; the REST and MCP system stats return goroutines/uptime; off-Linux the container fields read "unavailable".
- [ ] **Step 5: Pass + final gate + commit** `feat: add runtime cards to the dashboard`.

## Phase 7b Definition of Done

- `/metrics` serves Prometheus text and is excluded from recording.
- The runtime snapshot and `GET /api/v1/admin/stats/system` and `get_system_stats` return process figures plus container figures when available.
- The dashboard polls `/admin/runtime` every ~5s and shows unavailable fields honestly.
- Off-Linux degrades without error.
- Gates clean; docs updated.
