package forceexport

import (
	"fmt"
	"reflect"
	"runtime"
	"unsafe"
)

// GetFunc gets the function defined by the given fully-qualified name. The
// outFuncPtr parameter should be a pointer to a function with the appropriate
// type (e.g. the address of a local variable), and is set to a new function
// value that calls the specified function. If the specified function does not
// exist, outFuncPtr is not set and an error is returned.
func GetFunc(outFuncPtr interface{}, name string) error {
	codePtr, err := FindFuncWithName(name)
	if err != nil {
		return err
	}
	CreateFuncForCodePtr(outFuncPtr, codePtr)
	return nil
}

// Convenience struct for modifying the underlying code pointer of a function
// value. The actual struct has other values, but always starts with a code
// pointer.
type Func struct {
	codePtr uintptr
}

// CreateFuncForCodePtr is given a code pointer and creates a function value
// that uses that pointer. The outFun argument should be a pointer to a function
// of the proper type (e.g. the address of a local variable), and will be set to
// the result function value.
func CreateFuncForCodePtr(outFuncPtr interface{}, codePtr uintptr) {
	outFuncVal := reflect.ValueOf(outFuncPtr).Elem()
	// Use reflect.MakeFunc to create a well-formed function value that's
	// guaranteed to be of the right type and guaranteed to be on the heap
	// (so that we can modify it). We give a nil delegate function because
	// it will never actually be called.
	newFuncVal := reflect.MakeFunc(outFuncVal.Type(), nil)
	// Use reflection on the reflect.Value (yep!) to grab the underling
	// function value pointer. Trying to call newFuncVal.Pointer() wouldn't
	// work because it gives the code pointer rather than the function value
	// pointer. The function value is a struct that starts with its code
	// pointer, so we can swap out the code pointer with our desired value.
	funcPtr := (*Func)(unsafe.Pointer(reflect.ValueOf(newFuncVal).FieldByName("ptr").Pointer()))
	funcPtr.codePtr = codePtr
	outFuncVal.Set(newFuncVal)
}

type (

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

	bitvector struct {
		n        int32 // # of bits
		bytedata *uint8
	}

	moduleWrapper interface {
		GetFtab() []Functab
		GetFunc(ftab Functab) *runtime.Func
		GetNext() moduleWrapper
	}

	oldModuleWrapper struct {
		pclntable    []byte
		ftab         []Functab
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

	newModuleWrapper struct {
		pcHeader     *pcHeader
		funcnametab  []byte
		cutab        []uint32
		filetab      []byte
		pctab        []byte
		pclntable    []byte
		ftab         []Functab
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

	newModuleWrapper1_18 struct {
		pcHeader     *pcHeader1_18
		funcnametab  []byte
		cutab        []uint32
		filetab      []byte
		pctab        []byte
		pclntable    []byte
		ftab         []Functab
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

func (me *newModuleWrapper) GetFtab() []Functab {
	return me.ftab
}

func (me *newModuleWrapper) GetFunc(ftab Functab) *runtime.Func {
	return (*runtime.Func)(unsafe.Pointer(uintptr(unsafe.Pointer(me.pcHeader)) + uintptr(me.pcHeader.pclnOffset) + ftab.funcoff))
	//return (*runtime.Func)(unsafe.Pointer(&(*pcIntable)[ftab.funcoff]))
}

func (me *newModuleWrapper1_18) GetNext() moduleWrapper {
	if me.next != nil {
		return me.next
	}
	return nil
}

func (me *newModuleWrapper1_18) GetFtab() []Functab {
	return me.ftab
}

func (me *newModuleWrapper1_18) GetFunc(ftab Functab) *runtime.Func {
	ftab1_18 := (*Functab1_18)(unsafe.Pointer(&ftab))
	return (*runtime.Func)(unsafe.Pointer(uintptr(unsafe.Pointer(me.pcHeader)) + uintptr(me.pcHeader.pclnOffset) + uintptr(ftab1_18.funcoff)))
	//return (*runtime.Func)(unsafe.Pointer(&(*pcIntable)[ftab.funcoff]))
}

func (me *newModuleWrapper) GetNext() moduleWrapper {
	if me.next != nil {
		return me.next
	}
	return nil
}

func (me *oldModuleWrapper) GetFtab() []Functab {
	return me.ftab
}

func (me *oldModuleWrapper) GetFunc(ftab Functab) *runtime.Func {
	return (*runtime.Func)(unsafe.Pointer(&me.pclntable[ftab.funcoff]))
}

func (me *oldModuleWrapper) GetNext() moduleWrapper {
	if me.next != nil {
		return (*oldModuleWrapper)(unsafe.Pointer(me.next))
	}
	return nil
}

// FindFuncWithName searches through the moduledata table created by the linker
// and returns the function's code pointer. If the function was not found, it
// returns an error. Since the data structures here are not exported, we copy
// them below (and they need to stay in sync or else things will fail
// catastrophically).
func FindFuncWithName(name string) (uintptr, error) {
	var module moduleWrapper
	var new *newModuleWrapper
	var new1_18 *newModuleWrapper1_18
	var old *oldModuleWrapper

	if Firstmoduledata.pcHeader.magic == 0xFFFFFFF0 { // go 1.18+
		new1_18 = (*newModuleWrapper1_18)(unsafe.Pointer(&Firstmoduledata))
		module = new1_18
	} else if Firstmoduledata.pcHeader.magic == 0xFFFFFFFA { // go 1.16+
		new = (*newModuleWrapper)(unsafe.Pointer(&Firstmoduledata))
		// offset := uintptr(unsafe.Pointer(new)) - uintptr(unsafe.Pointer(new.pcHeader))
		// println(offset)
		// println(uintptr(unsafe.Pointer(&new.pclntable[0])) - uintptr(unsafe.Pointer(new.pcHeader)))
		// println(uintptr(unsafe.Pointer(&new.ftab[0])) - uintptr(unsafe.Pointer(new.pcHeader)))
		// println(uintptr(unsafe.Pointer(&new.pclntable[0])), uintptr(unsafe.Pointer(new)))
		// println(new.pcHeader.pclnOffset)
		module = new
	} else {
		old = (*oldModuleWrapper)(unsafe.Pointer(&Firstmoduledata))
		// println(&Firstmoduledata)
		// println(old)
		module = old
	}

	for {
		ftabs := module.GetFtab()
		l := len(ftabs)
		for i, ftab := range ftabs {
			if i == l-1 {
				break
			}
			f := module.GetFunc(ftab)
			n := f.Name()
			if n == name {
				return f.Entry(), nil
			}
		}
		module = module.GetNext()
		println(module)
		if module == nil {
			break
		}
	}

	return 0, fmt.Errorf("Invalid function name: %s", name)
}

// Everything below is taken from the runtime package, and must stay in sync
// with it.

//go:linkname Firstmoduledata runtime.firstmoduledata
var Firstmoduledata Moduledata

type Moduledata struct {
	pcHeader *pcHeader
}

type Moduledata1_18 struct {
	pcHeader *pcHeader1_18
}

type Functab struct {
	entry   uintptr
	funcoff uintptr
}

type Functab1_18 struct {
	entry   uint32
	funcoff uint32
}

type Bitvector struct {
	n        int32 // # of bits
	bytedata *uint8
}
