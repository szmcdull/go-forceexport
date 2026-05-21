//go:build go1.26
// +build go1.26

package forceexport

import (
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

// Go 1.26+ implementation. runtime.moduledata layout changed (epclntab, bad field order).
// Types must stay in sync with src/runtime/symtab.go.

type (
	pcHeader struct {
		magic          uint32 // 0xFFFFFFF1
		pad1, pad2     uint8
		minLC          uint8
		ptrSize        uint8
		nfunc          int
		nfiles         uint
		_              uintptr // was textStart; unused since Go 1.20+
		funcnameOffset uintptr
		cuOffset       uintptr
		filetabOffset  uintptr
		pctabOffset    uintptr
		pclnOffset     uintptr
	}

	newModuleWrapper moduledata

	Moduledata struct {
		pcHeader *pcHeader
	}

	Functab1_18 struct {
		entry   uint32
		funcoff uint32
	}

	initTask struct{} // slice element type only; full initTask is internal to runtime
)

type moduledata struct {
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
	covctrs, ecovctrs     uintptr
	end, gcdata, gcbss    uintptr
	types, etypes         uintptr
	rodata                uintptr
	gofunc                uintptr
	epclntab              uintptr

	textsectmap []textsect
	typelinks   []int32
	itablinks   []*itab

	ptab []ptabEntry

	pluginpath string
	pkghashes  []modulehash

	inittasks []*initTask

	modulename   string
	modulehashes []modulehash

	hasmain uint8
	bad     bool

	gcdatamask, gcbssmask bitvector

	typemap map[typeOff]*_type

	next *moduledata
}

type functab struct {
	entryoff uint32
	funcoff  uint32
}

type textsect struct {
	vaddr    uintptr
	end      uintptr
	baseaddr uintptr
}

type interfacetype struct {
	typ     _type
	pkgpath name
	mhdr    []imethod
}

type _type struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata     *byte
	str        nameOff
	ptrToThis  typeOff
}

type nameOff int32
type typeOff int32
type textOff int32

type ptabEntry struct {
	name nameOff
	typ  typeOff
}

type modulehash struct {
	modulename   string
	linktimehash string
	runtimehash  *string
}

type name struct {
	bytes *byte
}

type imethod struct {
	name nameOff
	ityp typeOff
}

type tflag uint8

type itab struct {
	Inter *interfacetype
	Type  *_type
	Hash  uint32
	Fun   [1]uintptr
}

func (me *newModuleWrapper) GetNext() moduleWrapper {
	if me.next != nil {
		return (*newModuleWrapper)(me.next)
	}
	return nil
}

func (me *newModuleWrapper) GetFtab() []functab {
	return me.ftab
}

func (me *newModuleWrapper) GetFunc(ftab functab) *runtime.Func {
	ftab1_18 := (*Functab1_18)(unsafe.Pointer(&ftab))
	return (*runtime.Func)(unsafe.Pointer(uintptr(unsafe.Pointer(me.pcHeader)) + uintptr(me.pcHeader.pclnOffset) + uintptr(ftab1_18.funcoff)))
}

func getModuleWrapper() moduleWrapper {
	if moduleDataAddr := findFirstModuleData(); moduleDataAddr != 0 {
		moduleData := (*moduledata)(unsafe.Pointer(moduleDataAddr))
		return (*newModuleWrapper)(unsafe.Pointer(moduleData))
	}
	return nil
}

var Firstmoduledata uintptr
var FirstmoduledataAddrFromLinkname uintptr
var firstModuleDataOnce sync.Once
var codeAddr uintptr

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
		pc := reflect.ValueOf(runtime.GC).Pointer()
		codeAddr = pc & ^uintptr(0xFFF)

		for offset := uintptr(0); offset < 0x2000000; offset += uintptr(unsafe.Sizeof(uintptr(0))) {
			if addr := codeAddr + offset; isValidModuleData(addr) {
				Firstmoduledata = addr
				return
			}

			if codeAddr > offset && codeAddr-offset > 0x400000 {
				if addr := codeAddr - offset; isValidModuleData(addr) {
					Firstmoduledata = addr
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

func isValidModuleData(addr uintptr) bool {
	if addr == 0 || addr < 0x1000 || addr == 0xffffffffffffffff {
		return false
	}

	if addr%uintptr(unsafe.Sizeof(uintptr(0))) != 0 {
		return false
	}

	pcHeaderPtrAddr := addr + 0
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

	const pcHeaderSize = 32
	if !IsAddrReadable(pcHeaderAddr, pcHeaderSize) {
		return false
	}

	magic, ok := safeReadUintptr(pcHeaderAddr)
	if !ok {
		return false
	}

	magicAndPads := magic & 0xffffffffffff
	if magicAndPads != 0xFFFFFFF1 && magicAndPads != 0xFFFFFFF0 {
		return false
	}

	nfunc, ok := safeReadUint32(pcHeaderAddr + 8)
	if !ok {
		return false
	}

	if nfunc == 0 || nfunc > 100000 {
		return false
	}

	moduleData := (*moduledata)(unsafe.Pointer(addr))
	if moduleData.hasmain != 1 {
		return false
	}

	return true
}

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
