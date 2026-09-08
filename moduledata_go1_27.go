//go:build go1.27
// +build go1.27

package forceexport

// Layout from Go 1.27.1 src/runtime/symtab.go. Type and itab metadata now
// use ranges instead of typelinks/itablinks slices. Keep field order in sync
// with runtime: discovery reads hasmain and function lookup follows next.
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

	text, etext                uintptr
	noptrdata, enoptrdata      uintptr
	data, edata                uintptr
	bss, ebss                  uintptr
	noptrbss, enoptrbss        uintptr
	covctrs, ecovctrs          uintptr
	end, gcdata, gcbss         uintptr
	types, typedesclen, etypes uintptr
	itaboffset, itabsize       uintptr
	rodata                     uintptr
	gofunc                     uintptr
	epclntab                   uintptr

	textsectmap []textsect

	ptab []ptabEntry

	pluginpath string
	pkghashes  []modulehash

	inittasks []*initTask

	modulename   string
	modulehashes []modulehash

	hasmain uint8
	bad     bool

	gcdatamask, gcbssmask bitvector

	typemap map[*_type]*_type

	next *moduledata
}
