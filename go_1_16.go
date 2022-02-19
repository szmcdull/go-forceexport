//go:build go1.16 && !go1.18
// +build go1.16,!go1.18

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

	newModuleWrapper struct {
		pcHeader     *pcHeader
		funcnametab  []byte
		cutab        []uint32
		filetab      []byte
		pctab        []byte
		pclntable    []byte
		ftab         []functab
		findfunctab  uintptr
		minpc, maxpc uintptr

		text, etext           uintptr
		noptrdata, enoptrdata uintptr
		data, edata           uintptr
		bss, ebss             uintptr
		noptrbss, enoptrbss   uintptr
		end, gcdata, gcbss    uintptr
		types, etypes         uintptr

		textsectmap []byte
		typelinks   []int32 // offsets from types
		itablinks   []byte

		ptab []byte

		pluginpath string
		pkghashes  []byte

		modulename   string
		modulehashes []byte

		hasmain uint8 // 1 if module contains the main function, 0 otherwise

		gcdatamask, gcbssmask bitvector

		typemap map[int32]*byte // offset to *_rtype in previous module

		bad bool // module failed to load and should be ignored

		next *newModuleWrapper
	}

	Moduledata struct {
		pcHeader *pcHeader
	}

	// pcHeader holds data used by the pclntab lookups.
	pcHeader struct {
		magic          uint32  // 0xFFFFFFFA
		pad1, pad2     uint8   // 0,0
		minLC          uint8   // min instruction size
		ptrSize        uint8   // size of a ptr in bytes
		nfunc          int     // number of functions in the module
		nfiles         uint    // number of entries in the file tab.
		funcnameOffset uintptr // offset to the funcnametab variable from pcHeader
		cuOffset       uintptr // offset to the cutab variable from pcHeader
		filetabOffset  uintptr // offset to the filetab variable from pcHeader
		pctabOffset    uintptr // offset to the pctab varible from pcHeader
		pclnOffset     uintptr // offset to the pclntab variable from pcHeader
	}

	// pcHeader holds data used by the pclntab lookups.
	pcHeader1_18 struct {
		magic          uint32  // 0xFFFFFFF0
		pad1, pad2     uint8   // 0,0
		minLC          uint8   // min instruction size
		ptrSize        uint8   // size of a ptr in bytes
		nfunc          int     // number of functions in the module
		nfiles         uint    // number of entries in the file tab
		textStart      uintptr // base for function entry PC offsets in this module, equal to moduledata.text
		funcnameOffset uintptr // offset to the funcnametab variable from pcHeader
		cuOffset       uintptr // offset to the cutab variable from pcHeader
		filetabOffset  uintptr // offset to the filetab variable from pcHeader
		pctabOffset    uintptr // offset to the pctab variable from pcHeader
		pclnOffset     uintptr // offset to the pclntab variable from pcHeader
	}

	newModuleWrapper1_18 struct {
		pcHeader     *pcHeader1_18
		funcnametab  []byte
		cutab        []uint32
		filetab      []byte
		pctab        []byte
		pclntable    []byte
		ftab         []functab
		findfunctab  uintptr
		minpc, maxpc uintptr

		text, etext           uintptr
		noptrdata, enoptrdata uintptr
		data, edata           uintptr
		bss, ebss             uintptr
		noptrbss, enoptrbss   uintptr
		end, gcdata, gcbss    uintptr
		types, etypes         uintptr
		rodata                uintptr
		gofunc                uintptr // go.func.*

		textsectmap []byte
		typelinks   []int32 // offsets from types
		itablinks   []byte

		ptab []byte

		pluginpath string
		pkghashes  []byte

		modulename   string
		modulehashes []byte

		hasmain uint8 // 1 if module contains the main function, 0 otherwise

		gcdatamask, gcbssmask bitvector

		typemap map[int32]*byte // offset to *_rtype in previous module

		bad bool // module failed to load and should be ignored

		next *newModuleWrapper1_18
	}
)

func (me *newModuleWrapper) GetFtab() []functab {
	return me.ftab
}

func (me *newModuleWrapper) GetFunc(ftab functab) *runtime.Func {
	return (*runtime.Func)(unsafe.Pointer(uintptr(unsafe.Pointer(me.pcHeader)) + uintptr(me.pcHeader.pclnOffset) + ftab.funcoff))
	//return (*runtime.Func)(unsafe.Pointer(&(*pcIntable)[ftab.funcoff]))
}

func (me *newModuleWrapper) GetNext() moduleWrapper {
	if me.next != nil {
		return me.next
	}
	return nil
}

func getModuleWrapper() moduleWrapper {
	new := (*newModuleWrapper)(unsafe.Pointer(&Firstmoduledata))
	return new
}

//go:linkname Firstmoduledata runtime.firstmoduledata
var Firstmoduledata Moduledata
