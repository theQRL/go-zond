// Package memsize provides utilities for calculating memory sizes
package memsize

import (
	"os"
	"runtime"
	"sync"
	"time"
)

// MemStats represents memory statistics
type MemStats struct {
	Alloc      uint64
	TotalAlloc uint64
	Sys        uint64
	NumGC      uint32
}

// GetMemoryUsage returns the current memory usage in bytes
func GetMemoryUsage() uint64 {
	// Get process memory info
	pid := os.Getpid()
	
	// Platform-specific implementations
	switch runtime.GOOS {
	case "windows":
		return getMemoryUsageWindows()
	case "linux":
		return getMemoryUsageLinux()
	case "darwin":
		return getMemoryUsageDarwin()
	default:
		// For other platforms, return a conservative estimate
		return 0
	}
}

// GetSystemMemoryUsage returns the total system memory usage in bytes
func GetSystemMemoryUsage() uint64 {
	// Platform-specific implementations
	switch runtime.GOOS {
	case "windows":
		return getSystemMemoryUsageWindows()
	case "linux":
		return getSystemMemoryUsageLinux()
	case "darwin":
		return getSystemMemoryUsageDarwin()
	default:
		// For other platforms, return a conservative estimate
		return 0
	}
}

// MonitorMemoryUsage starts monitoring memory usage and calls the callback
// function whenever memory usage changes significantly
func MonitorMemoryUsage(callback func(uint64), interval time.Duration) {
	var lastUsage uint64
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		currentUsage := GetMemoryUsage()
		if currentUsage != lastUsage {
			callback(currentUsage)
			lastUsage = currentUsage
		}
	}
}

// MemoryPool provides a thread-safe pool for memory operations
type MemoryPool struct {
	mu    sync.Mutex
	stats MemStats
}

// NewMemoryPool creates a new memory pool
func NewMemoryPool() *MemoryPool {
	return &MemoryPool{}
}

// UpdateStats updates the memory statistics
func (p *MemoryPool) UpdateStats() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Get current memory usage
	currentUsage := GetMemoryUsage()
	
	// Update stats with current values
	p.stats = MemStats{
		Alloc:      currentUsage,
		TotalAlloc: currentUsage, // For now, we'll use current usage as total
		Sys:        GetSystemMemoryUsage(),
		NumGC:      0, // We don't track GC count anymore
	}
}

// GetStats returns the current memory statistics
func (p *MemoryPool) GetStats() MemStats {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stats
} 