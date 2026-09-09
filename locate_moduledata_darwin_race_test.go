//go:build race && go1.23 && darwin
// +build race,go1.23,darwin

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

// TestMachOModuleDataScanUnderRace exercises the in-memory Mach-O writable
// segment scan under Go 1.26+ checkptr. Raw uintptr-to-pointer reads in that
// loop used to fatal before any test started when a dependency called GetFunc
// from init (for example go-cancelContext).
func TestMachOModuleDataScanUnderRace(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	previous := codeAddr
	codeAddr = pc & ^uintptr(0xfff)
	defer func() { codeAddr = previous }()

	addr := findModuleDataInMachOImage(pc)
	if addr == 0 {
		t.Fatal("findModuleDataInMachOImage returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}
}
