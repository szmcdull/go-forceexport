//go:build go1.23 && darwin && large_discovery_test
// +build go1.23,darwin,large_discovery_test

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

func touchLargeDiscoveryPadding()

func TestLargeDarwinModuleDataDiscovery(t *testing.T) {
	const legacyWindow = uintptr(32 << 20)
	touchLargeDiscoveryPadding()

	pc := reflect.ValueOf(runtime.GC).Pointer()
	codeAddr = pc & ^uintptr(0xfff)
	addr := findModuleDataInMachOImage(pc)
	if addr == 0 {
		t.Fatal("findModuleDataInMachOImage returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}

	distance := addr - pc
	if pc > addr {
		distance = pc - addr
	}
	if distance <= legacyWindow {
		t.Fatalf("fixture did not exceed the legacy window: runtime.GC=0x%x moduledata=0x%x distance=%d", pc, addr, distance)
	}
	t.Logf("runtime.GC=0x%x moduledata=0x%x distance=%d MiB", pc, addr, distance>>20)
}
