//go:build go1.23
// +build go1.23

package forceexport

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

// Go 1.23+ specific implementation: Due to linkname restrictions, use limited function discovery
// Reuse type definitions from go_1_21.go

// The first word of every non-empty interface type contains an *ITab.
// It records the underlying concrete type (Type), the interface type it
// is implementing (Inter), and some ancillary information.
//
// allocated in non-garbage-collected memory
type itab struct {
	Inter *interfacetype
	Type  *_type
	Hash  uint32     // copy of Type.Hash. Used for type switches.
	Fun   [1]uintptr // variable sized. fun[0]==0 means Type does not implement Inter.
}

type go123ModuleWrapper struct {
	funcs []functab
	next  *go123ModuleWrapper
}

func (m *go123ModuleWrapper) GetNext() moduleWrapper {
	if m.next != nil {
		return m.next
	}
	return nil
}

func (m *go123ModuleWrapper) GetFtab() []functab {
	return m.funcs
}

func (m *go123ModuleWrapper) GetFunc(ftab functab) *runtime.Func {
	pc := uintptr(ftab.entryoff)
	return runtime.FuncForPC(pc)
}

func getModuleWrapper() moduleWrapper {
	if moduleDataAddr := findFirstModuleData(); moduleDataAddr != 0 {
		// Found it! Handle in the same way as the old version
		moduleData := (*moduledata)(unsafe.Pointer(moduleDataAddr))
		return (*newModuleWrapper)(unsafe.Pointer(moduleData))
	}
	return nil
}

var Firstmoduledata uintptr
var FirstmoduledataAddrFromLinkname uintptr
var firstModuleDataOnce sync.Once
var codeAddr uintptr

// scan memory for runtime.firstmoduledata
func findFirstModuleData() uintptr {
	if Firstmoduledata != 0 {
		return Firstmoduledata
	}
	if FirstmoduledataAddrFromLinkname != 0 {
		return FirstmoduledataAddrFromLinkname
	}
	if FirstmoduledataFromLinkName.pcHeader != nil {
		FirstmoduledataAddrFromLinkname = uintptr(unsafe.Pointer(&FirstmoduledataFromLinkName))
		return FirstmoduledataAddrFromLinkname
	}

	firstModuleDataOnce.Do(func() {

		// Strategy 1: Locate nearby data structures using known runtime function addresses
		pc := reflect.ValueOf(runtime.GC).Pointer()

		// moduledata is usually in the data segment near the code segment
		// Search range: start from the current PC address, search forward and backward
		codeAddr = pc & ^uintptr(0xFFF) // Page aligned

		// Search for moduledata features within a reasonable range
		// Use a more conservative search range and step size
		for offset := uintptr(0); offset < 0x2000000; offset += uintptr(unsafe.Sizeof(uintptr(0))) { // Search 32MB range, step by pointer size
			// Search forward
			if addr := codeAddr + offset; isValidModuleData(addr) {
				Firstmoduledata = addr
				// fmt.Printf("Found moduledata at: %x, base addr: %x, offset=%x, FirstmoduledataFromLinkName=%x\n", addr, baseAddr, offset, FirstmoduledataAddrFromLinkname)
				return
			}

			// Search backward
			if codeAddr > offset && codeAddr-offset > 0x400000 { // Ensure not to search too low addresses
				if addr := codeAddr - offset; isValidModuleData(addr) {
					Firstmoduledata = addr
					// fmt.Printf("Found moduledata at: %x, base addr: %x, offset=%x, FirstmoduledataFromLinkName=%x\n", addr, baseAddr, offset, FirstmoduledataAddrFromLinkname)
					return
				}
			}
		}

	})

	return Firstmoduledata
}

func isInCodeSection(addr uintptr) bool {
	offset := int(addr) - int(codeAddr)
	if offset > 0x40000000 || offset < -200000 {
		return false
	}
	return true
}

// Check whether the given address is likely a moduledata structure
func isValidModuleData(addr uintptr) bool {
	// Basic address check
	if addr == 0 || addr < 0x1000 || addr == 0xffffffffffffffff {
		return false
	}

	// Alignment check - moduledata should be pointer aligned
	if addr%uintptr(unsafe.Sizeof(uintptr(0))) != 0 {
		return false
	}

	// Check whether the basic part of the moduledata structure can be safely read
	// const moduleDataMinSize = 64 // The first 64 bytes of moduledata contain key fields
	// if !IsAddrReadable(addr, int(unsafe.Sizeof(moduledata{}))) {
	// 	return false
	// }

	// Safely check the pcHeader pointer - pcHeader is usually the second field of moduledata
	pcHeaderPtrAddr := addr + 0 // Assume pcHeader is at offset 0
	// if !IsAddrReadable(pcHeaderPtrAddr, 8) {
	// 	return false
	// }

	// Try to safely read the pcHeader pointer
	IsAddrReadable(pcHeaderPtrAddr, int(unsafe.Sizeof(pcHeader{})))
	pcHeaderAddr, ok := safeReadUintptr(pcHeaderPtrAddr)
	if !ok {
		return false
	}

	if !isInCodeSection(pcHeaderAddr) {
		return false
	}

	if pcHeaderAddr == 0 || pcHeaderAddr < 0x1000 || pcHeaderAddr == 0xffffffffffffffff {
		return false
	}

	// Check whether pcHeader is readable
	const pcHeaderSize = 32 // Estimated size of pcHeader structure
	if !IsAddrReadable(pcHeaderAddr, pcHeaderSize) {
		return false
	}

	// Try to safely read the magic value
	magic, ok := safeReadUintptr(pcHeaderAddr)
	if !ok {
		return false
	}

	magicAndPads := magic & 0xffffffffffff
	if magicAndPads != 0xFFFFFFF1 && magicAndPads != 0xFFFFFFF0 {
		return false
	}

	// Try to safely read the nfunc field
	nfunc, ok := safeReadUint32(pcHeaderAddr + 8) // nfunc usually follows magic
	if !ok {
		return false
	}

	if nfunc == 0 || nfunc > 100000 { // Reasonable function count range
		return false
	}

	moduleData := (*moduledata)(unsafe.Pointer(addr))
	if moduleData.hasmain != 1 {
		return false
	}

	// If all checks pass, consider this a valid moduledata
	return true
}

// Safely read uintptr value
func safeReadUintptr(addr uintptr) (value uintptr, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
			value = 0
		}
	}()

	if !IsAddrReadable(addr, int(unsafe.Sizeof(uintptr(0)))) {
		return 0, false
	}

	value = *(*uintptr)(unsafe.Pointer(addr))
	ok = true
	return
}

// Safely read uint32 value
func safeReadUint32(addr uintptr) (value uint32, ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
			value = 0
		}
	}()

	if !IsAddrReadable(addr, 4) {
		return 0, false
	}

	value = *(*uint32)(unsafe.Pointer(addr))
	ok = true
	return
}
