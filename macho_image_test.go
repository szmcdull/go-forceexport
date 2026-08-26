//go:build go1.23
// +build go1.23

package forceexport

import (
	"encoding/binary"
	"testing"
)

func TestParseMachO64(t *testing.T) {
	data := make([]byte, 32+2*72)
	put32 := func(off int, value uint32) { binary.LittleEndian.PutUint32(data[off:], value) }
	put64 := func(off int, value uint64) { binary.LittleEndian.PutUint64(data[off:], value) }
	put32(0, machOMagic64)
	put32(12, machOTypeExecutable)
	put32(16, 2)
	put32(20, 2*72)

	put32(32, machOLoadSegment64)
	put32(36, 72)
	put64(32+24, 0x100000000)
	put64(32+32, 0x200000)
	put64(32+40, 0)
	put64(32+48, 0x200000)
	put32(32+60, machOVMProtRead|machOVMProtExecute)

	off := 32 + 72
	put32(off, machOLoadSegment64)
	put32(off+4, 72)
	put64(off+24, 0x100200000)
	put64(off+32, 0x100000)
	put64(off+40, 0x200000)
	put64(off+48, 0x80000)
	put32(off+60, machOVMProtRead|machOVMProtWrite)

	fileType, segments, ok := parseMachO64(data)
	if !ok {
		t.Fatal("parseMachO64 rejected a valid image")
	}
	if fileType != machOTypeExecutable || len(segments) != 2 {
		t.Fatalf("fileType=%d segments=%d", fileType, len(segments))
	}
	if got := segments[1]; got.vmaddr != 0x100200000 || got.vmsize != 0x100000 || got.initprot != machOVMProtRead|machOVMProtWrite {
		t.Fatalf("unexpected data segment: %+v", got)
	}
}

func TestParseMachO64RejectsMalformedCommands(t *testing.T) {
	data := make([]byte, 32+72)
	binary.LittleEndian.PutUint32(data[0:], machOMagic64)
	binary.LittleEndian.PutUint32(data[12:], machOTypeExecutable)
	binary.LittleEndian.PutUint32(data[16:], 1)
	binary.LittleEndian.PutUint32(data[20:], 72)
	binary.LittleEndian.PutUint32(data[32:], machOLoadSegment64)
	binary.LittleEndian.PutUint32(data[36:], 80)
	if _, _, ok := parseMachO64(data); ok {
		t.Fatal("parseMachO64 accepted a command extending past sizeofcmds")
	}
}

func TestLoadedMachOImageRangesAppliesSlide(t *testing.T) {
	segments := []machOSegmentSpec{
		{vmaddr: 0, vmsize: 0x100000000}, // __PAGEZERO is not mapped.
		{vmaddr: 0x100000000, vmsize: 0x200000, fileoff: 0, filesize: 0x200000, initprot: machOVMProtRead | machOVMProtExecute},
		{vmaddr: 0x100200000, vmsize: 0x100000, fileoff: 0x200000, filesize: 0x80000, initprot: machOVMProtRead | machOVMProtWrite},
	}
	ranges, ok := loadedMachOImageRanges(0x40000000, segments)
	if !ok || len(ranges) != 2 {
		t.Fatalf("ok=%v ranges=%+v", ok, ranges)
	}
	if ranges[0].start != 0x40000000 || ranges[0].end != 0x40200000 {
		t.Fatalf("unexpected text range: %+v", ranges[0])
	}
	if ranges[1].start != 0x40200000 || ranges[1].end != 0x40300000 {
		t.Fatalf("unexpected data range: %+v", ranges[1])
	}
}
