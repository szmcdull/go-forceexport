//go:build go1.23 && linux
// +build go1.23,linux

package forceexport

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

type procMapEntry struct {
	start, end uintptr
	perms      string
	pathname   string
}

// findModuleDataInProcessMaps locates runtime.firstmoduledata by scanning the
// main executable's read-write mappings. moduledata lives in .go.module, which
// is placed in the data segment after .text/.rodata.
func findModuleDataInProcessMaps(codeAddr uintptr) uintptr {
	rwSegments := executableRWSegmentsAfter(codeAddr)
	for _, seg := range rwSegments {
		if addr := scanSegmentForModuleData(seg.start, seg.end); addr != 0 {
			return addr
		}
	}
	return 0
}

func executableRWSegmentsAfter(anchorPC uintptr) []procMapEntry {
	entries, err := parseProcMaps()
	if err != nil || len(entries) == 0 {
		return nil
	}

	var exePath string
	var exeEnd uintptr
	found := false
	for _, e := range entries {
		if anchorPC >= e.start && anchorPC < e.end && strings.Contains(e.perms, "x") {
			exePath = e.pathname
			exeEnd = e.end
			found = true
			break
		}
	}
	if !found || exePath == "" {
		return nil
	}

	var rw []procMapEntry
	for _, e := range entries {
		if e.pathname != exePath {
			continue
		}
		if !strings.Contains(e.perms, "w") || strings.Contains(e.perms, "x") {
			continue
		}
		if e.start < exeEnd {
			continue
		}
		rw = append(rw, e)
	}
	return rw
}

func parseProcMaps() ([]procMapEntry, error) {
	data, err := os.ReadFile("/proc/self/maps")
	if err != nil {
		return nil, err
	}

	var entries []procMapEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		start, end, ok := parseMapAddressRange(fields[0])
		if !ok {
			continue
		}
		pathname := ""
		if len(fields) > 5 {
			pathname = fields[len(fields)-1]
		}
		entries = append(entries, procMapEntry{
			start:    start,
			end:      end,
			perms:    fields[1],
			pathname: pathname,
		})
	}
	return entries, scanner.Err()
}

func parseMapAddressRange(field string) (start, end uintptr, ok bool) {
	parts := strings.Split(field, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	start64, err := strconv.ParseUint(parts[0], 16, 64)
	if err != nil {
		return 0, 0, false
	}
	end64, err := strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return 0, 0, false
	}
	return uintptr(start64), uintptr(end64), true
}

func scanSegmentForModuleData(start, end uintptr) uintptr {
	if start == 0 || end <= start {
		return 0
	}

	step := uintptr(unsafe.Sizeof(uintptr(0)))
	limit := end
	if limit > start+step {
		limit -= step
	}
	for addr := start; addr <= limit; addr += step {
		if isValidModuleData(addr) {
			return addr
		}
	}
	return 0
}
