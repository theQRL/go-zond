//go:build windows
// +build windows

package memsize

import (
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetProcessInfo = kernel32.NewProc("GetProcessMemoryInfo")
)

type PROCESS_MEMORY_COUNTERS struct {
	cb                         uint32
	PageFaultCount            uint32
	PeakWorkingSetSize        uint64
	WorkingSetSize           uint64
	QuotaPeakPagedPoolUsage  uint64
	QuotaPagedPoolUsage      uint64
	QuotaPeakNonpagedPoolUsage uint64
	QuotaNonpagedPoolUsage   uint64
	PagefileUsage            uint64
	PeakPagefileUsage        uint64
}

// getMemoryUsageWindows returns the current memory usage in bytes for Windows
func getMemoryUsageWindows() uint64 {
	var memCounters PROCESS_MEMORY_COUNTERS
	memCounters.cb = uint32(unsafe.Sizeof(memCounters))
	
	handle := syscall.GetCurrentProcess()
	ret, _, _ := procGetProcessInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&memCounters)),
		uintptr(memCounters.cb),
	)
	
	if ret == 0 {
		return 0
	}
	
	return memCounters.WorkingSetSize
}

// getSystemMemoryUsageWindows returns the total system memory usage in bytes for Windows
func getSystemMemoryUsageWindows() uint64 {
	var memInfo struct {
		dwLength        uint32
		dwMemoryLoad    uint32
		ullTotalPhys    uint64
		ullAvailPhys    uint64
		ullTotalPageFile uint64
		ullAvailPageFile uint64
		ullTotalVirtual  uint64
		ullAvailVirtual  uint64
		ullAvailExtendedVirtual uint64
	}
	
	memInfo.dwLength = uint32(unsafe.Sizeof(memInfo))
	
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGlobalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")
	
	ret, _, _ := procGlobalMemoryStatusEx.Call(
		uintptr(unsafe.Pointer(&memInfo)),
	)
	
	if ret == 0 {
		return 0
	}
	
	return memInfo.ullTotalPhys - memInfo.ullAvailPhys
} 