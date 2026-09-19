package sysinfo

import (
	"strconv"
	"strings"
	"time"
)

type Stats struct {
	Available  bool
	CPUPercent float64
	MemUsed    uint64
	MemLimit   uint64
	DiskUsed   uint64
	DiskTotal  uint64
}

func parseCPUmax(s string) (quota, period int64, unlimited bool) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, 0, false
	}
	period, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	if fields[0] == "max" {
		return 0, period, true
	}
	quota, err = strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, period, false
	}
	return quota, period, false
}

func statsAvailable(cpuOK, memOK, diskOK bool) bool {
	return cpuOK || memOK || diskOK
}

func cpuPercent(deltaUsec int64, wall time.Duration) float64 {
	if wall.Microseconds() <= 0 || deltaUsec <= 0 {
		return 0
	}
	return float64(deltaUsec) / float64(wall.Microseconds()) * 100
}
