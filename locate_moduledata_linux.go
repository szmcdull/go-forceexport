//go:build go1.23 && linux
// +build go1.23,linux

package forceexport

func locateModuleDataWithoutLinkname(codeAddr uintptr) uintptr {
	return findModuleDataInProcessMaps(codeAddr)
}
