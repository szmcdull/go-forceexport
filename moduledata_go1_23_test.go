//go:build go1.23 && !go1.26
// +build go1.23,!go1.26

package forceexport

import (
	"testing"
	"unsafe"
)

func TestGo123ModuleDataNextOffset(t *testing.T) {
	var m moduledata
	if got, want := unsafe.Offsetof(m.bad), unsafe.Offsetof(m.hasmain)+1; got != want {
		t.Fatalf("bad offset = %d, want hasmain+1 = %d", got, want)
	}
	if got, want := unsafe.Offsetof(m.next), unsafe.Offsetof(m.typemap)+unsafe.Sizeof(m.typemap); got != want {
		t.Fatalf("next offset = %d, want end of typemap = %d", got, want)
	}

	wrapper, ok := getModuleWrapper().(*newModuleWrapper)
	if !ok || wrapper == nil {
		t.Fatal("moduledata not found")
	}
	if wrapper.hasmain != 1 || wrapper.bad {
		t.Fatalf("unexpected main module flags: hasmain=%d bad=%v", wrapper.hasmain, wrapper.bad)
	}
	if wrapper.next != nil {
		t.Fatalf("main module next = %p, want nil", wrapper.next)
	}
}
