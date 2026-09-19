//go:build !linux

package sysinfo

func Read(dataDir string) (Stats, error) {
	return Stats{Available: false}, nil
}
