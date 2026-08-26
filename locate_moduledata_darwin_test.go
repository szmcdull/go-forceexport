//go:build go1.23 && darwin
// +build go1.23,darwin

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

func TestFindModuleDataInMachOImage(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	codeAddr = pc & ^uintptr(0xfff)
	addr := findModuleDataInMachOImage(pc)
	if addr == 0 {
		t.Fatal("findModuleDataInMachOImage returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}

	_, ranges := findMachOImage(pc)
	for _, r := range ranges {
		if r.prot&machOVMProtWrite != 0 && addr >= r.start && addr < r.end {
			return
		}
	}
	t.Fatalf("moduledata 0x%x is not in a writable Mach-O segment", addr)
}
