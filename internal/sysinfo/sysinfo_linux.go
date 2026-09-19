//go:build linux

package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func readUsageUsec() (int64, bool) {
	body, err := os.ReadFile("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return 0, false
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "usage_usec" {
			v, err := strconv.ParseInt(fields[1], 10, 64)
			return v, err == nil
		}
	}
	return 0, false
}

func readMemory() (used, limit uint64) {
	if body, err := os.ReadFile("/sys/fs/cgroup/memory.current"); err == nil {
		used, _ = strconv.ParseUint(strings.TrimSpace(string(body)), 10, 64)
	}
	if body, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		v := strings.TrimSpace(string(body))
		if v != "max" {
			limit, _ = strconv.ParseUint(v, 10, 64)
		}
	}
	return used, limit
}

func readDisk(dataDir string) (used, total uint64) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &fs); err != nil {
		return 0, 0
	}
	total = fs.Blocks * uint64(fs.Bsize)
	free := fs.Bavail * uint64(fs.Bsize)
	return total - free, total
}

func Read(dataDir string) (Stats, error) {
	st := Stats{Available: true}

	if first, ok := readUsageUsec(); ok {
		time.Sleep(200 * time.Millisecond)
		if second, ok := readUsageUsec(); ok {
			st.CPUPercent = cpuPercent(second-first, 200*time.Millisecond)
		}
	}

	st.MemUsed, st.MemLimit = readMemory()
	st.DiskUsed, st.DiskTotal = readDisk(dataDir)
	return st, nil
}
