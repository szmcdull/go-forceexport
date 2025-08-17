//go:build windows

package forceexport

import (
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procVirtualQuery = kernel32.NewProc("VirtualQuery")
	procIsBadReadPtr = kernel32.NewProc("IsBadReadPtr")
)

const (
	MEM_COMMIT    = 0x1000
	PAGE_NOACCESS = 0x01
	PAGE_GUARD    = 0x100
)

type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
}

// IsAddrReadable checks whether a memory address is readable using Windows API
func IsAddrReadable(addr uintptr, size int) bool {
	if addr == 0 || size <= 0 {
		return false
	}

	// Basic range check
	if addr < 0x10000 || addr == 0xffffffffffffffff {
		return false
	}

	// Check for overflow
	if addr+uintptr(size) < addr {
		return false
	}

	// Use VirtualQuery to check memory state
	var mbi MEMORY_BASIC_INFORMATION
	ret, _, _ := procVirtualQuery.Call(
		addr,
		uintptr(unsafe.Pointer(&mbi)),
		unsafe.Sizeof(mbi),
	)

	if ret == 0 {
		return false
	}

	// Check if memory is committed and readable
	if mbi.State != MEM_COMMIT {
		return false
	}

	// Check protection attributes
	if mbi.Protect&PAGE_NOACCESS != 0 || mbi.Protect&PAGE_GUARD != 0 {
		return false
	}

	// Ensure the entire region is within this memory block
	if addr+uintptr(size) > mbi.BaseAddress+mbi.RegionSize {
		return false
	}

	return true
}
