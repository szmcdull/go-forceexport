//go:build go1.26 && !checklinkname_off
// +build go1.26,!checklinkname_off

package forceexport

import _ "unsafe"

// go 1.26 and above: when checklinkname is on (the default), forceexport will search the memory for runtime.firstmoduledata.
// This may take a few seconds. See go_1_26_checklinkname_on.go for faster boot.
var FirstmoduledataFromLinkName Moduledata
