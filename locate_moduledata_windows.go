//go:build go1.23 && windows
// +build go1.23,windows

package forceexport

import (
	"sort"
	"unsafe"
)

const (
	imageDOSSignature = 0x5a4d
	imageNTSignature  = 0x00004550

	imageScnMemExecute = 0x20000000
	imageScnMemRead    = 0x40000000
	imageScnMemWrite   = 0x80000000
)

type imageSectionRange struct {
	start uintptr
	end   uintptr
}

// locateModuleDataWithoutLinkname locates firstmoduledata in the writable
// sections of the main executable image. Go's linker puts moduledata in the
// data portion of the image, which can be much farther than 32 MiB from an
// arbitrary text symbol in a large executable.
func locateModuleDataWithoutLinkname(codeAddr uintptr) uintptr {
	return findModuleDataInPEImage(codeAddr)
}

func findModuleDataInPEImage(codeAddr uintptr) uintptr {
	base, imageEnd, sections := executableWritableSections(codeAddr)
	if base == 0 || imageEnd <= base {
		return 0
	}

	// Linker-generated moduledata is normally late in the image. Search higher
	// sections and addresses first, and only perform the expensive full
	// validation after the pcHeader magic prefilter matches.
	sort.Slice(sections, func(i, j int) bool { return sections[i].end > sections[j].end })
	for _, section := range sections {
		if addr := scanPESegmentForModuleData(section.start, section.end, base, imageEnd); addr != 0 {
			return addr
		}
	}
	return 0
}

func executableWritableSections(codeAddr uintptr) (base, imageEnd uintptr, sections []imageSectionRange) {
	var mbi MEMORY_BASIC_INFORMATION
	ret, _, _ := procVirtualQuery.Call(codeAddr, uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
	if ret == 0 || mbi.AllocationBase == 0 {
		return 0, 0, nil
	}
	base = mbi.AllocationBase

	if !IsAddrReadable(base, 0x40) || *(*uint16)(unsafe.Pointer(base)) != imageDOSSignature {
		return 0, 0, nil
	}
	ntOffset := uintptr(*(*uint32)(unsafe.Pointer(base + 0x3c)))
	if ntOffset > 1<<20 || base+ntOffset < base || !IsAddrReadable(base+ntOffset, 24) {
		return 0, 0, nil
	}
	nt := base + ntOffset
	if *(*uint32)(unsafe.Pointer(nt)) != imageNTSignature {
		return 0, 0, nil
	}

	numberOfSections := uintptr(*(*uint16)(unsafe.Pointer(nt + 6)))
	sizeOfOptionalHeader := uintptr(*(*uint16)(unsafe.Pointer(nt + 20)))
	if numberOfSections == 0 || numberOfSections > 96 || sizeOfOptionalHeader < 60 {
		return 0, 0, nil
	}
	optionalHeader := nt + 24
	if optionalHeader < nt || !IsAddrReadable(optionalHeader, int(sizeOfOptionalHeader)) {
		return 0, 0, nil
	}
	sizeOfImage := uintptr(*(*uint32)(unsafe.Pointer(optionalHeader + 56)))
	if sizeOfImage == 0 || base+sizeOfImage < base {
		return 0, 0, nil
	}
	imageEnd = base + sizeOfImage

	sectionTable := optionalHeader + sizeOfOptionalHeader
	sectionTableSize := numberOfSections * 40
	if sectionTable < optionalHeader || !IsAddrReadable(sectionTable, int(sectionTableSize)) {
		return 0, 0, nil
	}
	for i := uintptr(0); i < numberOfSections; i++ {
		header := sectionTable + i*40
		virtualSize := uintptr(*(*uint32)(unsafe.Pointer(header + 8)))
		virtualAddress := uintptr(*(*uint32)(unsafe.Pointer(header + 12)))
		characteristics := *(*uint32)(unsafe.Pointer(header + 36))
		if virtualSize == 0 || characteristics&imageScnMemRead == 0 ||
			characteristics&imageScnMemWrite == 0 || characteristics&imageScnMemExecute != 0 {
			continue
		}
		start := base + virtualAddress
		end := start + virtualSize
		if start < base || end < start || start >= imageEnd {
			continue
		}
		if end > imageEnd {
			end = imageEnd
		}
		sections = append(sections, imageSectionRange{start: start, end: end})
	}
	return base, imageEnd, sections
}

func scanPESegmentForModuleData(start, end, imageBase, imageEnd uintptr) uintptr {
	step := uintptr(unsafe.Sizeof(uintptr(0)))
	if start == 0 || end <= start || end-start < step {
		return 0
	}
	addr := (end - step) & ^(step - 1)
	start = (start + step - 1) & ^(step - 1)
	for addr >= start {
		pcHeaderAddr := *(*uintptr)(unsafe.Pointer(addr))
		if pcHeaderAddr >= imageBase && pcHeaderAddr+8 >= pcHeaderAddr && pcHeaderAddr+8 <= imageEnd {
			magicAndPads := *(*uintptr)(unsafe.Pointer(pcHeaderAddr)) & 0xffffffffffff
			if magicAndPads == 0xFFFFFFF1 || magicAndPads == 0xFFFFFFF0 {
				if isValidModuleData(addr) {
					return addr
				}
			}
		}
		if addr < start+step {
			break
		}
		addr -= step
	}
	return 0
}
