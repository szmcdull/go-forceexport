//go:build race && go1.23

package forceexport

import "testing"

// TestModuleDataScanUnderRace verifies that runtime.firstmoduledata memory scan
// does not fatal under checkptr (-race). Run with:
//
//	CGO_ENABLED=1 go test -race -run TestModuleDataScanUnderRace
func TestModuleDataScanUnderRace(t *testing.T) {
	addr := findFirstModuleData()
	if addr == 0 {
		t.Fatal("findFirstModuleData returned 0")
	}

	var addOne func(int) int
	if err := GetFunc(&addOne, "github.com/szmcdull/go-forceexport.addOne"); err != nil {
		t.Fatal(err)
	}
	if addOne(2) != 3 {
		t.Fatalf("addOne(2) = %d, want 3", addOne(2))
	}
}
