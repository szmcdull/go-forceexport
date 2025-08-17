package forceexport

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func init() {
	fmt.Println(time.Now())
	c, cancel := context.WithCancel(context.Background())
	fmt.Println(c, cancel)
}

// This test fails when not debugging, even though checklinkname=0.
// This may be due to the optimization done by the Go compiler (withContext inlined?).
func TestContext(t *testing.T) {
	c, cancel := context.WithCancel(context.Background())
	cancel()
	c.Done()

	var v func(int) int
	err := GetFunc(&v, `context.withCancel`)
	if err != nil {
		t.Error("Expected nil error.")
	}
	if v == nil {
		t.Error("Expected non-nil function.")
	}
}

func TestTimeNow(t *testing.T) {
	// Try the new time.runtimeNow first (3 return values) - based on call stack
	var runtimeNowFunc func() (int64, int32, int64)
	err := GetFunc(&runtimeNowFunc, "time.runtimeNow")
	if err == nil && runtimeNowFunc != nil {
		sec, nsec, mono := runtimeNowFunc()
		if sec == 0 || nsec == 0 {
			t.Error("Expected nonzero result from time.runtimeNow().")
		}
		t.Logf("time.runtimeNow() returned sec=%d, nsec=%d, mono=%d", sec, nsec, mono)
		return
	}

	// Fallback to assembly time.now (2 return values)
	var timeNowFunc func() (int64, int32)
	err = GetFunc(&timeNowFunc, "time.now")
	if err == nil && timeNowFunc != nil {
		sec, nsec := timeNowFunc()
		if sec == 0 || nsec == 0 {
			t.Error("Expected nonzero result from time.now().")
		}
		t.Logf("time.now() returned sec=%d, nsec=%d", sec, nsec)
		return
	}

	// If both fail, report the failure
	t.Fatalf("Failed to get both time.runtimeNow and time.now: %v", err)
}

// Note that we need to disable inlining here, or else the function won't be
// compiled into the binary. We also need to call it from the test so that the
// compiler doesn't remove it because it's unused.
//
//go:noinline
func addOne(x int) int {
	return x + 1
}

func TestAddOne(t *testing.T) {
	if addOne(3) != 4 {
		t.Error("addOne should work properly.")
	}

	var addOneFunc func(x int) int
	err := GetFunc(&addOneFunc, "github.com/szmcdull/go-forceexport.addOne")
	if err != nil {
		t.Error("Expected nil error.")
	}
	if addOneFunc(3) != 4 {
		t.Error("Expected addOneFunc to add one to 3.")
	}
}

func GetPointer(v interface{}) uintptr {
	val := reflect.ValueOf(v)
	return val.Pointer()
}

func TestFunc1(t *testing.T) {
}

func TestFunc2(t *testing.T) {
}

func TestFunc3(t *testing.T) {
}

func TestFunc4(t *testing.T) {
}

func TestInvalidFunc(t *testing.T) {
	var invalidFunc func()
	err := GetFunc(&invalidFunc, "invalidpackage.invalidfunction")
	if err == nil {
		t.Error("Expected an error.")
	}
	if invalidFunc != nil {
		t.Error("Expected a nil function.")
	}
}

func TestForceExport(t *testing.T) {
	var func1, func2, func3, func4 func(*testing.T)
	_ = GetFunc(&func1, `github.com/szmcdull/go-forceexport.TestFunc1`)
	_ = GetFunc(&func2, `github.com/szmcdull/go-forceexport.TestFunc2`)
	_ = GetFunc(&func3, `github.com/szmcdull/go-forceexport.TestFunc3`)
	_ = GetFunc(&func4, `github.com/szmcdull/go-forceexport.TestFunc4`)
	if func1 == nil || func2 == nil || func3 == nil || func4 == nil {
		t.Error(`func == nil`)
	} else {
		// r1 := func1()
		// r2 := func2()
		// r3 := func3()
		// r4 := func4()
		// if r1 != 1 || r2 != 2 || r3 != 3 || r4 != 4 {
		// 	t.Error(`result wrong`)
		// }
	}
}
