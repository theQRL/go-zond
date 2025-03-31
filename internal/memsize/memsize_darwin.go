//go:build darwin
// +build darwin

package memsize

import (
	"syscall"
	"unsafe"
)

type mach_task_basic_info struct {
	Reserved       uint32
	VirtualSize    uint64
	ResidentSize   uint64
	ResidentSizeMax uint64
	UserTime       uint64
	SystemTime     uint64
	Policy         int32
	SuspendCount   int32
}

type host_vm_info64_t struct {
	FreeSize           uint64
	ActiveSize         uint64
	InactiveSize       uint64
	WireSize           uint64
	ZeroFillSize       uint64
	Reactivations      uint64
	Pageins            uint64
	Pageouts           uint64
	Faults             uint64
	CowFaults          uint64
	Lookups            uint64
	Hits               uint64
	Purges             uint64
	PurgeableCount     uint32
	SpeculativeCount   uint32
	Decompressions     uint64
	Compressions       uint64
	Swapins            uint64
	Swapouts           uint64
	CompressorPageCount uint64
	ThrottledCount     uint64
	ExternalPageCount  uint64
	InternalPageCount  uint64
	TotalUncompressedPagesInCompressor uint64
}

var (
	libc = syscall.NewLazyDLL("libc.dylib")
	procTaskInfo = libc.NewProc("task_info")
	procHostInfo = libc.NewProc("host_statistics64")
)

// getMemoryUsageDarwin returns the current memory usage in bytes for macOS
func getMemoryUsageDarwin() uint64 {
	var info mach_task_basic_info
	infoSize := uint32(unsafe.Sizeof(info))
	
	handle := syscall.GetCurrentProcess()
	ret, _, _ := procTaskInfo.Call(
		uintptr(handle),
		uintptr(syscall.TASK_BASIC_INFO),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&infoSize)),
	)
	
	if ret != 0 {
		return 0
	}
	
	return info.ResidentSize
}

// getSystemMemoryUsageDarwin returns the total system memory usage in bytes for macOS
func getSystemMemoryUsageDarwin() uint64 {
	var info host_vm_info64_t
	infoSize := uint32(unsafe.Sizeof(info))
	
	ret, _, _ := procHostInfo.Call(
		uintptr(syscall.HOST_VM_INFO64),
		uintptr(unsafe.Pointer(&info)),
		uintptr(unsafe.Pointer(&infoSize)),
	)
	
	if ret != 0 {
		return 0
	}
	
	return info.ActiveSize + info.WireSize
} 