//go:build race && go1.23

package forceexport

import "testing"

// initAddOne is resolved during package init, matching downstream libraries
// such as go-cancelContext. Go 1.26+ enables checkptr under -race; a fatal
// here aborts the process before any test function runs.
var initAddOne func(int) int

func init() {
	_ = GetFunc(&initAddOne, "github.com/szmcdull/go-forceexport.addOne")
}

// TestGetFuncDuringInitUnderRace verifies that GetFunc during package init
// survives checkptr. This is not a data race: -race only enables stricter
// unsafe-pointer checks. Temporary workaround if an older forceexport is
// still in use:
//
//	go test -race -gcflags=all=-d=checkptr=0
func TestGetFuncDuringInitUnderRace(t *testing.T) {
	if initAddOne == nil {
		t.Fatal("GetFunc during init left initAddOne nil")
	}
	if got := initAddOne(2); got != 3 {
		t.Fatalf("initAddOne(2) = %d, want 3", got)
	}
}

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
