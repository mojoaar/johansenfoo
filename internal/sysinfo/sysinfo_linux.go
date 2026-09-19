//go:build linux

package sysinfo

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	cgroupRoot = "/sys/fs/cgroup"
	procRoot   = "/proc"
)

func readUsageUsec() (int64, bool) {
	body, err := os.ReadFile(cgroupRoot + "/cpu.stat")
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

func readCPUQuota() (quota, period int64, unlimited bool) {
	body, err := os.ReadFile(cgroupRoot + "/cpu.max")
	if err != nil {
		return 0, 0, true
	}
	return parseCPUmax(string(body))
}

func readMemory() (used, limit uint64, ok bool) {
	if body, err := os.ReadFile(cgroupRoot + "/memory.current"); err == nil {
		used, err = strconv.ParseUint(strings.TrimSpace(string(body)), 10, 64)
		ok = err == nil
	}
	if body, err := os.ReadFile(cgroupRoot + "/memory.max"); err == nil {
		v := strings.TrimSpace(string(body))
		if v != "max" {
			limit, err = strconv.ParseUint(v, 10, 64)
			ok = ok || err == nil
		}
	}
	if ok {
		return used, limit, true
	}
	return readProcMemory()
}

func readProcMemory() (used, limit uint64, ok bool) {
	body, err := os.ReadFile(procRoot + "/meminfo")
	if err != nil {
		return 0, 0, false
	}
	var total, available uint64
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			total = value * 1024
		case "MemAvailable":
			available = value * 1024
		}
	}
	if total == 0 {
		return 0, 0, false
	}
	return total - available, total, true
}

func readDisk(dataDir string) (used, total uint64, ok bool) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(dataDir, &fs); err != nil {
		return 0, 0, false
	}
	total = fs.Blocks * uint64(fs.Bsize)
	free := fs.Bavail * uint64(fs.Bsize)
	return total - free, total, true
}

func Read(dataDir string) (Stats, error) {
	var st Stats

	cpuOK := false
	if first, ok := readUsageUsec(); ok {
		time.Sleep(200 * time.Millisecond)
		if second, ok := readUsageUsec(); ok {
			percent := cpuPercent(second-first, 200*time.Millisecond)
			if quota, period, unlimited := readCPUQuota(); !unlimited && quota > 0 && period > 0 {
				percent = percent * float64(period) / float64(quota)
			}
			st.CPUPercent = percent
			cpuOK = true
		}
	}

	memOK := false
	st.MemUsed, st.MemLimit, memOK = readMemory()

	diskUsed, diskTotal, diskOK := readDisk(dataDir)
	st.DiskUsed, st.DiskTotal = diskUsed, diskTotal

	st.Available = statsAvailable(cpuOK, memOK, diskOK)
	return st, nil
}
