//go:build go1.23 && linux
// +build go1.23,linux

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestFindModuleDataInProcessMaps(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	codeAddr := pc & ^uintptr(0xFFF)

	addr := findModuleDataInProcessMaps(codeAddr)
	if addr == 0 {
		t.Fatal("findModuleDataInProcessMaps returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}

	inRW := false
	for _, seg := range executableRWSegmentsAfter(pc) {
		if addr >= seg.start && addr < seg.end {
			inRW = true
			break
		}
	}
	if !inRW {
		t.Fatalf("moduledata 0x%x is not in executable rw segments", addr)
	}
}

func TestFindFuncUsesProcessMaps(t *testing.T) {
	// time.runtimeNow does not exist until Go 1.24. time.Now is present on
	// every supported version and uses the normal Go calling convention.
	var now func() time.Time
	if err := GetFunc(&now, "time.Now"); err != nil {
		t.Fatalf("GetFunc(time.Now): %v", err)
	}
	if now == nil {
		t.Fatal("time.Now is nil")
	}
	if got := now(); got.IsZero() {
		t.Fatal("time.Now returned zero time")
	}
}
