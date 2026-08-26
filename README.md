# go-forceexport

go-forceexport is a golang package that allows access to any module-level
function, even ones that are not exported. You give it the string name of a
function , like `"time.now"`, and gives you a function value that calls that
function. More generally, it can be used to achieve something like reflection on
top-level functions, whereas the `reflect` package only lets you access methods
by name.

As you might expect, this library is **unsafe** and **fragile** and probably
shouldn't be used in production. See "Use cases and pitfalls" below.

Tested on **Linux** (Go 1.23–1.26, including Delve `-gcflags="all=-N -l"` builds)
and on **macOS** with older Go versions. On Go 1.23+ without `checklinkname_off`:

- **Linux** — maps-based scan of the executable RW segment (recommended path)
- **macOS** — in-memory Mach-O load-command scan of writable segments
- **Windows / other** — legacy GC-anchored scan (±32 MiB); use
  `checklinkname_off` if that is not enough

## Installation

`$ go get github.com/szmcdull/go-forceexport`

## Usage

Here's how you can grab the `time.now` function, defined as
`func now() (sec int64, nsec int32)`

```go
var timeNow func() (int64, int32)
err := forceexport.GetFunc(&timeNow, "time.now")
if err != nil {
    // Handle errors if you care about name possibly being invalid.
}
// Calls the actual time.now function.
sec, nsec := timeNow()
```

The string you give should be the fully-qualified name. For example, here's
`GetFunc` getting itself.

```go
var getFunc func(interface{}, string) error
GetFunc(&getFunc, "github.com/alangpierce/go-forceexport.GetFunc")
```

## The following Go versions are tested:
- 1.26.1
- 1.25
- 1.23
- 1.21
- 1.18 beta2
- 1.17.5
- 1.16.13
- 1.14.15

### Note for Go 1.23 and above

Due to the restriction to `go:linkname` in recent Go versions, you can compile with
`-tags=checklinkname_off -ldflags=-checklinkname=0` to enable `go:linkname` and let
forceexport link directly to `runtime.firstmoduledata` (fastest, most reliable).

If you cannot require those flags on downstream consumers (e.g. a library used by
others), forceexport falls back to a **runtime scan** to find `runtime.firstmoduledata`
in memory. The scan strategy is selected by OS at compile time; see
**Runtime moduledata discovery** below.

## Runtime moduledata discovery (Go 1.23+)

`GetFunc` walks `runtime.firstmoduledata` to resolve function names. On Go 1.23+ with default
`checklinkname=1`, forceexport cannot link directly to that symbol and must locate it first.

### Two paths

| Path | Build flags | Behaviour |
|------|-------------|-----------|
| **linkname** | `-tags=checklinkname_off -ldflags=-checklinkname=0` | `go:linkname runtime.firstmoduledata` |
| **runtime scan** | default | OS-specific; see platform table |

### Platform dispatch

Entry point: `locateModuleDataWithoutLinkname` (called from `go_1_23.go` / `go_1_26.go`).
The version-specific files do **not** embed scan logic themselves.

| OS | Files | Strategy |
|----|-------|----------|
| **Linux** | `locate_moduledata_linux.go` → `moduledata_maps_linux.go` | `/proc/self/maps` RW-segment scan |
| **macOS** | `locate_moduledata_darwin.go` → `macho_image.go` | in-memory Mach-O writable-segment scan |
| **other** | `locate_moduledata_legacy.go` | scan ±32 MiB around `runtime.GC` |

### Linux: `/proc/self/maps` RW-segment scan

On Linux, `runtime.firstmoduledata` lives in the linker-generated **`.go.module`** section. In the
loaded image this sits in the executable's **read-write data segment**, after `.text` / `.rodata`.

Algorithm (`moduledata_maps_linux.go`):

1. Take `runtime.GC` PC and find the containing **r-x** mapping in `/proc/self/maps`.
2. Collect **rw-** mappings of the **same executable path** with start address ≥ end of that r-x map.
3. Walk only those RW regions and return the first address that passes `isValidModuleData`.

Why not a fixed window from `runtime.GC`? On large **Delve / debug** binaries
(`-gcflags="all=-N -l"`), `.go.module` can sit **beyond any practical fixed limit**
(e.g. >32 MiB above code). Scanning the whole RW segment avoids both missed lookups and
slow blind scans over tens of megabytes of code/rodata.

Typical virtual layout (Linux PIE):

```
[ .text r-x ] [ .rodata r-- ] [ .go.module / .data rw- ]
   runtime.GC here              runtime.firstmoduledata here
```

### macOS: Mach-O, not ELF

macOS executables use **Mach-O**, not ELF. A Go binary is usually **Mach-O 64-bit**
(`MH_EXECUTE` or `MH_PIE`). Rough segment mapping:

| Mach-O segment | Role | ELF analogue |
|----------------|------|--------------|
| `__TEXT` | code, read-only metadata | `.text` |
| `__DATA_CONST` | read-only globals | `.rodata` |
| `__DATA` | writable globals | `.data`, `.go.module`, `.noptrdata`, … |

`runtime.firstmoduledata` is still in a **writable non-executable segment after code**, but
there is **no `/proc/self/maps`**. The Darwin implementation finds the in-memory Mach-O header
from the `runtime.GC` anchor, parses its `LC_SEGMENT_64` commands, applies the ASLR slide, and
scans writable non-executable segments. A pcHeader-magic prefilter avoids full validation for
nearly every pointer-sized slot. It is cgo-free and supports both amd64 and arm64 builds.

The format parser and ASLR calculations are platform-independent unit tests. The end-to-end
test in `locate_moduledata_darwin_test.go` must run on a macOS host or CI runner; cross-compiling
it on Linux verifies both Darwin architectures but cannot execute Mach syscalls.

### Windows

Not covered by the Linux maps scanner. Use `checklinkname_off` or the legacy / Windows-specific
paths in this repo where applicable.

### Troubleshooting

| Error | Likely cause | Fix |
|-------|--------------|-----|
| `moduledata not found!` | runtime scan failed before any function lookup | Linux: should not happen on recent builds; report a bug. Other OS: try `checklinkname_off` |
| `Invalid function name: …` | moduledata found, but symbol missing (inlining / dead-code elimination) | call a reference to keep the symbol; build with `-gcflags="all=-l"`; or use `checklinkname_off` |
| panic in `go-cancelContext` init | same as `moduledata not found!` during `GetFunc` in `init()` | ensure forceexport version with Linux maps scan, or add `checklinkname_off` to debug build flags |

## Use cases and pitfalls

This library is most useful for development and hack projects. For example, you
might use it to track down why the standard library isn't behaving as you
expect, or you might use it to try out a standard library function to see if it
works, then later factor the code to be less fragile. You could also try using
it in production; just make sure you're aware of the risks.

There are lots of things to watch out for and ways to shoot yourself in
the foot:
* If you define the wrong function type, you'll get a function with undefined
  behavior that will likely cause a runtime panic. The library makes no attempt
  to warn you in this case.
* Calling unexported functions is inherently fragile because the function won't
  have any stability guarantees.
* The implementation relies on the details of internal Go data structures, so
  later versions of Go might break this library.
* Since the compiler doesn't expect unexported symbols to be used, it might not
  create them at all, for example due to inlining or dead code analysis. This
  means that functions may not show up like you expect, and new versions of the
  compiler may cause functions to suddenly disappear.
* If the function you want to use relies on unexported types, you won't be able
  to trivially use it. However, you can sometimes work around this by defining
  equivalent copies of those types that you can use, but that approach has its
  own set of dangers.

## How it works

1. **Locate `runtime.firstmoduledata`**
   - With `checklinkname_off`: direct `go:linkname` to the linker symbol.
   - Otherwise: `locateModuleDataWithoutLinkname` (Linux maps scan, Darwin Mach-O scan, or
     legacy GC scan).

2. **Walk the function table** — same idea as `runtime.FuncForPC`: iterate `moduledata` entries
   until the requested name matches, then read the code pointer.

3. **Build a callable `func` value** — `reflect.MakeFunc` plus `unsafe.Pointer` to swap in the
   target code pointer.

Needless to say, it's a scary hack, but it seems to work!

## License

MIT
