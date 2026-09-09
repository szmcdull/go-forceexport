//go:build race && go1.23 && linux
// +build race,go1.23,linux

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

// TestProcessMapsModuleDataScanUnderRace exercises the /proc/self/maps
// read-write segment scan under checkptr, matching the Darwin/Windows race
// tests for their image scanners.
func TestProcessMapsModuleDataScanUnderRace(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	previous := codeAddr
	codeAddr = pc & ^uintptr(0xfff)
	defer func() { codeAddr = previous }()

	addr := findModuleDataInProcessMaps(codeAddr)
	if addr == 0 {
		t.Fatal("findModuleDataInProcessMaps returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}
}
