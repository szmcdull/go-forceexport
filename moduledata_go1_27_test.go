//go:build go1.27
// +build go1.27

package forceexport

import "testing"

// Exercise the fields that moved in Go 1.27 in both discovery and linkname
// builds. Linkname bypasses isValidModuleData, so function lookup alone does
// not check the metadata between ftab and next.
func TestGo127ModuleData(t *testing.T) {
	m, ok := getModuleWrapper().(*newModuleWrapper)
	if !ok || m == nil {
		t.Fatal("moduledata not found")
	}
	if m.hasmain != 1 || m.bad {
		t.Fatalf("unexpected main module flags: hasmain=%d bad=%v", m.hasmain, m.bad)
	}
	if m.types == 0 || m.etypes <= m.types {
		t.Fatalf("invalid type range: [%#x, %#x)", m.types, m.etypes)
	}
	size := m.etypes - m.types
	if m.typedesclen == 0 || m.typedesclen > size {
		t.Fatalf("invalid type descriptor size: %d (type range %d)", m.typedesclen, size)
	}
	if m.itaboffset < m.typedesclen || m.itaboffset > size || m.itabsize > size-m.itaboffset {
		t.Fatalf("invalid itab range: offset=%d size=%d (type range %d)", m.itaboffset, m.itabsize, size)
	}
}
