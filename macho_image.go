//go:build go1.23
// +build go1.23

package forceexport

import "encoding/binary"

const (
	machOMagic64        = 0xfeedfacf
	machOLoadSegment64  = 0x19
	machOTypeExecutable = 0x2
	machOTypeDylib      = 0x6
	machOTypeBundle     = 0x8

	machOVMProtRead    = 0x1
	machOVMProtWrite   = 0x2
	machOVMProtExecute = 0x4
)

type machOSegmentSpec struct {
	vmaddr   uint64
	vmsize   uint64
	fileoff  uint64
	filesize uint64
	initprot uint32
}

type machOImageRange struct {
	start uintptr
	end   uintptr
	prot  uint32
}

// parseMachO64 parses only the fixed-size fields needed for in-memory image
// discovery. Keeping this byte-slice parser platform independent lets its
// bounds and overflow handling be tested without a Darwin host.
func parseMachO64(data []byte) (fileType uint32, segments []machOSegmentSpec, ok bool) {
	if len(data) < 32 || binary.LittleEndian.Uint32(data[0:4]) != machOMagic64 {
		return 0, nil, false
	}

	fileType = binary.LittleEndian.Uint32(data[12:16])
	switch fileType {
	case machOTypeExecutable, machOTypeDylib, machOTypeBundle:
	default:
		return 0, nil, false
	}

	ncmds := binary.LittleEndian.Uint32(data[16:20])
	sizeofcmds := binary.LittleEndian.Uint32(data[20:24])
	if ncmds == 0 || ncmds > 4096 || sizeofcmds > 16<<20 || uint64(32)+uint64(sizeofcmds) > uint64(len(data)) {
		return 0, nil, false
	}

	off := uint64(32)
	commandsEnd := off + uint64(sizeofcmds)
	for i := uint32(0); i < ncmds; i++ {
		if off+8 > commandsEnd {
			return 0, nil, false
		}
		cmd := binary.LittleEndian.Uint32(data[off : off+4])
		cmdsize := binary.LittleEndian.Uint32(data[off+4 : off+8])
		if cmdsize < 8 || cmdsize&7 != 0 || off+uint64(cmdsize) > commandsEnd {
			return 0, nil, false
		}
		if cmd == machOLoadSegment64 {
			if cmdsize < 72 {
				return 0, nil, false
			}
			nsects := binary.LittleEndian.Uint32(data[off+64 : off+68])
			if uint64(nsects) > (uint64(cmdsize)-72)/80 {
				return 0, nil, false
			}
			segments = append(segments, machOSegmentSpec{
				vmaddr:   binary.LittleEndian.Uint64(data[off+24 : off+32]),
				vmsize:   binary.LittleEndian.Uint64(data[off+32 : off+40]),
				fileoff:  binary.LittleEndian.Uint64(data[off+40 : off+48]),
				filesize: binary.LittleEndian.Uint64(data[off+48 : off+56]),
				initprot: binary.LittleEndian.Uint32(data[off+60 : off+64]),
			})
		}
		off += uint64(cmdsize)
	}
	if off != commandsEnd || len(segments) == 0 {
		return 0, nil, false
	}
	return fileType, segments, true
}

func loadedMachOImageRanges(base uintptr, segments []machOSegmentSpec) ([]machOImageRange, bool) {
	var textVMAddr uint64
	foundText := false
	for _, seg := range segments {
		if seg.fileoff == 0 && seg.filesize != 0 && seg.initprot&machOVMProtExecute != 0 {
			textVMAddr = seg.vmaddr
			foundText = true
			break
		}
	}
	if !foundText {
		return nil, false
	}

	ranges := make([]machOImageRange, 0, len(segments))
	for _, seg := range segments {
		if seg.vmsize == 0 || seg.vmaddr < textVMAddr {
			continue
		}
		delta := seg.vmaddr - textVMAddr
		if delta > uint64(^uintptr(0)) || seg.vmsize > uint64(^uintptr(0)) {
			return nil, false
		}
		start := base + uintptr(delta)
		if start < base {
			return nil, false
		}
		end := start + uintptr(seg.vmsize)
		if end < start {
			return nil, false
		}
		ranges = append(ranges, machOImageRange{start: start, end: end, prot: seg.initprot})
	}
	return ranges, len(ranges) != 0
}
