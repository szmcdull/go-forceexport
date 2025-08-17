//go:build !windows && !linux && !darwin && !unix

package forceexport

import (
	"unsafe"
)

// IsAddrReadable fallback实现，用于不支持的平台
// 使用保守的地址范围检查，避免直接内存访问
func IsAddrReadable(addr uintptr, size int) bool {
	if addr == 0 || size <= 0 {
		return false
	}

	// 基本范围检查
	if addr < 0x1000 || addr == 0xffffffffffffffff {
		return false
	}

	// 检查溢出
	if addr+uintptr(size) < addr {
		return false
	}

	// 非常保守的地址检查，只允许明显安全的地址范围
	return fallbackAddressCheck(addr, size)
}

// 回退方案的地址检查
func fallbackAddressCheck(addr uintptr, size int) bool {
	if unsafe.Sizeof(uintptr(0)) == 8 { // 64位系统
		// 只允许非常安全的地址范围
		// 典型程序的代码段和数据段
		if addr >= 0x400000 && addr <= 0x10000000 {
			return true
		}

		// 典型的堆地址范围
		if addr >= 0x20000000 && addr <= 0x80000000 {
			return true
		}
	} else { // 32位系统
		// 32位系统的安全范围
		if addr >= 0x10000 && addr <= 0x40000000 {
			return true
		}
	}

	return false
}
