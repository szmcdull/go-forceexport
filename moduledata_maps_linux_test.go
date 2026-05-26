//go:build go1.23 && linux
// +build go1.23,linux

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
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
	var runtimeNow func() (int64, int32, int64)
	if err := GetFunc(&runtimeNow, "time.runtimeNow"); err != nil {
		t.Fatalf("GetFunc(time.runtimeNow): %v", err)
	}
	if runtimeNow == nil {
		t.Fatal("time.runtimeNow is nil")
	}
	sec, nsec, _ := runtimeNow()
	if sec == 0 || nsec == 0 {
		t.Fatalf("time.runtimeNow returned zero timestamp: sec=%d nsec=%d", sec, nsec)
	}
}