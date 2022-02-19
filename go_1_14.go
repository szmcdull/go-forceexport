//go:build !go1.16
// +build !go1.16

package forceexport

import (
	"runtime"
	"unsafe"
)

type (
	functab struct {
		entry   uintptr
		funcoff uintptr
	}

	oldModuleWrapper struct {
		pclntable    []byte
		ftab         []functab
		filetab      []uint32
		findfunctab  uintptr
		minpc, maxpc uintptr

		text, etext           uintptr
		noptrdata, enoptrdata uintptr
		data, edata           uintptr
		bss, ebss             uintptr
		noptrbss, enoptrbss   uintptr
		end, gcdata, gcbss    uintptr

		// Original type was []*_type
		typelinks []interface{}

		modulename string
		// Original type was []modulehash
		modulehashes []interface{}

		gcdatamask, gcbssmask Bitvector

		next *oldModuleWrapper
	}

	Bitvector struct {
		n        int32 // # of bits
		bytedata *uint8
	}
)

func (me *oldModuleWrapper) GetFtab() []functab {
	return me.ftab
}

func (me *oldModuleWrapper) GetFunc(ftab functab) *runtime.Func {
	return (*runtime.Func)(unsafe.Pointer(&me.pclntable[ftab.funcoff]))
}

func (me *oldModuleWrapper) GetNext() moduleWrapper {
	if me.next != nil {
		return (*oldModuleWrapper)(unsafe.Pointer(me.next))
	}
	return nil
}

func getModuleWrapper() moduleWrapper {
	old := &Firstmoduledata
	// println(&Firstmoduledata)
	// println(old)
	return old
}

//go:linkname Firstmoduledata runtime.firstmoduledata
var Firstmoduledata oldModuleWrapper
