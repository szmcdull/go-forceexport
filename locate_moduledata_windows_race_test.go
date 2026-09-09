//go:build race && go1.23 && windows
// +build race,go1.23,windows

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

// TestPEModuleDataScanUnderRace exercises the PE writable-section scan under
// checkptr. Same constraint as the Darwin Mach-O scanner: uintptr-to-pointer
// reads must not fatal when -race is enabled.
func TestPEModuleDataScanUnderRace(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	previous := codeAddr
	codeAddr = pc & ^uintptr(0xfff)
	defer func() { codeAddr = previous }()

	addr := findModuleDataInPEImage(pc)
	if addr == 0 {
		t.Fatal("findModuleDataInPEImage returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}
}
