//go:build go1.26 && checklinkname_off
// +build go1.26,checklinkname_off

package forceexport

import _ "unsafe"

// go 1.26 and above: to take the advantage of go:linkname, you must compile with
// -tags=checklinkname_off -ldflags=-checklinkname=0
//
//go:linkname FirstmoduledataFromLinkName runtime.firstmoduledata
var FirstmoduledataFromLinkName Moduledata
