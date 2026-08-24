//go:build go1.23 && !linux && !windows
// +build go1.23,!linux,!windows

package forceexport

import "unsafe"

func locateModuleDataWithoutLinkname(codeAddr uintptr) uintptr {
	for offset := uintptr(0); offset < 0x2000000; offset += uintptr(unsafe.Sizeof(uintptr(0))) {
		if addr := codeAddr + offset; isValidModuleData(addr) {
			return addr
		}

		if codeAddr > offset && codeAddr-offset > 0x400000 {
			if addr := codeAddr - offset; isValidModuleData(addr) {
				return addr
			}
		}
	}
	return 0
}
