//go:build go1.23 && darwin
// +build go1.23,darwin

package forceexport

import (
	"sort"
	"syscall"
	"unsafe"
)

func locateModuleDataWithoutLinkname(codeAddr uintptr) uintptr {
	return findModuleDataInMachOImage(codeAddr)
}

// findModuleDataInMachOImage locates firstmoduledata in the main Mach-O
// image. Unlike the legacy search, its reach is determined by load commands,
// so a large __TEXT or __DATA segment does not impose an arbitrary 32 MiB
// limit.
func findModuleDataInMachOImage(codeAddr uintptr) uintptr {
	_, ranges := findMachOImage(codeAddr)
	if len(ranges) == 0 {
		return 0
	}

	var writable []machOImageRange
	for _, r := range ranges {
		if r.prot&machOVMProtWrite != 0 && r.prot&machOVMProtExecute == 0 {
			writable = append(writable, r)
		}
	}
	// Go places moduledata in __DATA, after __DATA_CONST. Prefer later
	// writable segments, while scanning each one from its beginning.
	sort.Slice(writable, func(i, j int) bool { return writable[i].start > writable[j].start })
	for _, r := range writable {
		if addr := scanMachOSegmentForModuleData(r.start, r.end, ranges); addr != 0 {
			return addr
		}
	}
	return 0
}

func findMachOImage(codeAddr uintptr) (uintptr, []machOImageRange) {
	pageSize := uintptr(syscall.Getpagesize())
	if codeAddr == 0 || pageSize == 0 {
		return 0, nil
	}

	// A Go runtime text symbol is close to the beginning of __TEXT even when
	// application text is huge. Scan page-aligned candidates backwards to the
	// Mach-O header, then let the header validate that the anchor belongs to an
	// executable segment. The cap also bounds work for malformed mappings.
	const maxHeaderDistance = uintptr(256 << 20)
	start := codeAddr & ^(pageSize - 1)
	for distance := uintptr(0); distance <= maxHeaderDistance && distance <= start; distance += pageSize {
		base := start - distance
		magic, ok := safeReadUint32(base)
		if !ok || magic != machOMagic64 {
			continue
		}
		ranges, ok := machOImageRanges(base)
		if !ok {
			continue
		}
		for _, r := range ranges {
			if r.prot&machOVMProtExecute != 0 && codeAddr >= r.start && codeAddr < r.end {
				return base, ranges
			}
		}
	}
	return 0, nil
}

//go:nocheckptr
func machOImageRanges(base uintptr) ([]machOImageRange, bool) {
	const headerSize = uintptr(32)
	if !IsAddrReadable(base, int(headerSize)) {
		return nil, false
	}
	sizeofcmds := uintptr(*(*uint32)(unsafe.Pointer(base + 20)))
	if sizeofcmds == 0 || sizeofcmds > 16<<20 || base+headerSize+sizeofcmds < base {
		return nil, false
	}
	total := headerSize + sizeofcmds
	if !IsAddrReadable(base, int(total)) {
		return nil, false
	}
	data := unsafe.Slice((*byte)(unsafe.Pointer(base)), int(total))
	_, segments, ok := parseMachO64(data)
	if !ok {
		return nil, false
	}

	return loadedMachOImageRanges(base, segments)
}

func scanMachOSegmentForModuleData(start, end uintptr, imageRanges []machOImageRange) uintptr {
	step := uintptr(unsafe.Sizeof(uintptr(0)))
	moduleSize := uintptr(unsafe.Sizeof(moduledata{}))
	start = (start + step - 1) & ^(step - 1)
	if start == 0 || end <= start || end-start < moduleSize {
		return 0
	}
	limit := end - moduleSize
	for addr := start; addr <= limit; addr += step {
		pcHeaderAddr := *(*uintptr)(unsafe.Pointer(addr))
		if !rangeContains(imageRanges, pcHeaderAddr, step) {
			continue
		}
		magicAndPads := *(*uintptr)(unsafe.Pointer(pcHeaderAddr)) & 0xffffffffffff
		if (magicAndPads == 0xFFFFFFF1 || magicAndPads == 0xFFFFFFF0) && isValidModuleData(addr) {
			return addr
		}
	}
	return 0
}

func rangeContains(ranges []machOImageRange, addr, size uintptr) bool {
	if addr == 0 || addr+size < addr {
		return false
	}
	for _, r := range ranges {
		if r.prot&machOVMProtRead != 0 && addr >= r.start && addr+size <= r.end {
			return true
		}
	}
	return false
}
