//go:build go1.21 && !go1.23
// +build go1.21,!go1.23

// For Go 1.21-1.22: can use go:linkname normally

package forceexport

import "unsafe"

// layout of Itab known to compilers
// allocated in non-garbage-collected memory
// Needs to be in sync with
// ../cmd/compile/internal/reflectdata/reflect.go:/^func.WriteTabs.
type itab struct {
	inter *interfacetype
	_type *_type
	hash  uint32 // copy of _type.hash. Used for type switches.
	_     [4]byte
	fun   [1]uintptr // variable sized. fun[0]==0 means _type does not implement inter.
}

// go 1.23 and above: must compile with -ldflags=-checklinkname=0
//
//go:linkname Firstmoduledata runtime.firstmoduledata
var Firstmoduledata Moduledata

func getModuleWrapper() moduleWrapper {
	new1_18 := (*newModuleWrapper)(unsafe.Pointer(&Firstmoduledata))
	return new1_18
}
