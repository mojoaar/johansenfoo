package sysinfo

import (
	"testing"
	"time"
)

func TestParseCPUmax(t *testing.T) {
	quota, period, unlimited := parseCPUmax("200000 100000")
	if unlimited || quota != 200000 || period != 100000 {
		t.Fatalf("got quota=%d period=%d unlimited=%v", quota, period, unlimited)
	}
	quota, period, unlimited = parseCPUmax("max 100000")
	if !unlimited || period != 100000 {
		t.Fatalf("got quota=%d period=%d unlimited=%v, want unlimited", quota, period, unlimited)
	}
	if _, _, unlimited := parseCPUmax("garbage"); unlimited {
		t.Fatal("garbage should not parse as unlimited")
	}
}

func TestCPUPercent(t *testing.T) {
	if got := cpuPercent(500000, time.Second); got != 50 {
		t.Errorf("cpuPercent = %v, want 50", got)
	}
	if got := cpuPercent(0, time.Second); got != 0 {
		t.Errorf("cpuPercent(0) = %v, want 0", got)
	}
	if got := cpuPercent(1000, 0); got != 0 {
		t.Errorf("cpuPercent with no wall time = %v, want 0", got)
	}
}

func TestReadDegradesWithoutError(t *testing.T) {
	st, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !st.Available && (st.CPUPercent != 0 || st.MemUsed != 0 || st.DiskTotal != 0) {
		t.Errorf("unavailable stats must be zeroed: %+v", st)
	}
}

func TestAvailableHelper(t *testing.T) {
	if !statsAvailable(true, false, false) {
		t.Error("one successful read should be available")
	}
	if statsAvailable(false, false, false) {
		t.Error("no successful read should be unavailable")
	}
}

func TestCPUPercentSubMicrosecondWall(t *testing.T) {
	if got := cpuPercent(1000, 500*time.Nanosecond); got != 0 {
		t.Errorf("cpuPercent = %v, want 0 for a sub-microsecond window", got)
	}
}
