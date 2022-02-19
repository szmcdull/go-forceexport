package forceexport

import (
	"runtime"
	_ "unsafe"
)

type (
	bitvector struct {
		n        int32 // # of bits
		bytedata *uint8
	}

	// Functab struct {
	// 	entry   uintptr
	// 	funcoff uintptr
	// }

	moduleWrapper interface {
		GetFtab() []functab
		GetFunc(ftab functab) *runtime.Func
		GetNext() moduleWrapper
	}
)

// //go:linkname Firstmoduledata runtime.firstmoduledata
// var Firstmoduledata Moduledata
