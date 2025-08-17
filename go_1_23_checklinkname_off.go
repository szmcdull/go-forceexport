//go:build go1.23 && !checklinkname_off
// +build go1.23,!checklinkname_off

package forceexport

import _ "unsafe"

// go 1.23 and above: when checklinkname is on (the default), forceexport will search the memory for runtime.firstmoduledata.
// This may take a few seconds. See go_1_23_checklinkname_off.go for faster boot.
var FirstmoduledataFromLinkName Moduledata
