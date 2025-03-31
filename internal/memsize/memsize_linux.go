//go:build linux
// +build linux

package memsize

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// getMemoryUsageLinux returns the current memory usage in bytes for Linux
func getMemoryUsageLinux() uint64 {
	file, err := os.Open("/proc/self/statm")
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 2 {
		return 0
	}

	// The second field is the resident set size in pages
	rss, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	// Convert pages to bytes (assuming 4KB pages)
	return rss * 4096
}

// getSystemMemoryUsageLinux returns the total system memory usage in bytes for Linux
func getSystemMemoryUsageLinux() uint64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer file.Close()

	var total, free uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				total, _ = strconv.ParseUint(fields[1], 10, 64)
				total *= 1024 // Convert KB to bytes
			}
		} else if strings.HasPrefix(line, "MemFree:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				free, _ = strconv.ParseUint(fields[1], 10, 64)
				free *= 1024 // Convert KB to bytes
			}
		}
	}

	return total - free
} 