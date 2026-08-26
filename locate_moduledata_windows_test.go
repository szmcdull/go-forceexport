//go:build go1.23 && windows
// +build go1.23,windows

package forceexport

import (
	"reflect"
	"runtime"
	"testing"
)

func TestFindModuleDataInPEImage(t *testing.T) {
	pc := reflect.ValueOf(runtime.GC).Pointer()
	addr := findModuleDataInPEImage(pc)
	if addr == 0 {
		t.Fatal("findModuleDataInPEImage returned 0")
	}
	if !isValidModuleData(addr) {
		t.Fatalf("invalid moduledata at 0x%x", addr)
	}

	_, _, sections := executableWritableSections(pc)
	for _, section := range sections {
		if addr >= section.start && addr < section.end {
			return
		}
	}
	t.Fatalf("moduledata 0x%x is not in a writable PE section", addr)
}
