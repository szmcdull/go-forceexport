//go:build linux || darwin || unix

package forceexport

import (
	"syscall"
	"unsafe"
)

// IsAddrReadable checks whether a memory address is readable using Unix system calls
func IsAddrReadable(addr uintptr, size int) bool {
	if addr == 0 || size <= 0 {
		return false
	}

	// Basic range check
	if addr < 0x1000 || addr == 0xffffffffffffffff {
		return false
	}

	// Check for overflow
	if addr+uintptr(size) < addr {
		return false
	}

	// Try to use msync to check if the memory is readable
	// msync will return ENOMEM if the memory region is invalid
	pageSize := uintptr(syscall.Getpagesize())

	// Align to page boundary
	pageStart := addr & ^(pageSize - 1)
	pageEnd := ((addr + uintptr(size) + pageSize - 1) & ^(pageSize - 1))

	// Use msync to check if the memory page is valid
	// This is safer than directly accessing memory
	_, _, errno := syscall.Syscall(syscall.SYS_MSYNC, pageStart, pageEnd-pageStart, syscall.MS_ASYNC)
	if errno != 0 {
		// If msync fails, the memory may be invalid
		if errno == syscall.ENOMEM || errno == syscall.EFAULT {
			return false
		}
	}

	// Further conservative check
	return conservativeUnixAddressCheck(addr, size)
}

// Conservative address check for Unix systems
func conservativeUnixAddressCheck(addr uintptr, size int) bool {
	if unsafe.Sizeof(uintptr(0)) == 8 { // 64-bit system
		// Typical user space layout for Linux/Unix 64-bit systems
		// Program segment usually starts near 0x400000
		// Heap is usually at lower addresses, stack at higher addresses

		// Check if in a reasonable program address range
		if addr >= 0x400000 && addr <= 0x7fffffffffff {
			return true
		}

		// Check if in a typical heap address range
		if addr >= 0x1000000 && addr <= 0x40000000000 {
			return true
		}
	} else { // 32-bit system
		// 32-bit systems have a smaller address space
		if addr >= 0x8000 && addr <= 0x7fffffff {
			return true
		}
	}

	return false
}
