package node

import (
	"fmt"
	"path"
	"runtime"
	"strings"
)

// GetDirAndFile retrieves the file and line number of the caller.
// Depth is the skip-caller count.  Clues funcs that call this one should
// provide either `1` (if they do not already have a depth value), or `depth+1`
// otherwise`.
//
// formats:
// dir `absolute/os/path/to/parent/folder`
// fileAndLine `<file>:<line>`
// parentAndFileAndLine `<parent>/<file>:<line>`
func GetDirAndFile(
	depth int,
) (dir, fileAndLine, parentAndFileAndLine string) {
	_, file, line, _ := runtime.Caller(depth + 1)
	dir, file = path.Split(file)

	fileLine := fmt.Sprintf("%s:%d", file, line)
	parentFileLine := fileLine

	parent := path.Base(dir)
	if len(parent) > 0 {
		parentFileLine = path.Join(parent, fileLine)
	}

	return dir, fileLine, parentFileLine
}

// GetCaller retrieves the func name of the caller. Depth is the  skip-caller
// count.  Clues funcs that call this one should provide either `1` (if they
// do not already have a depth value), or `depth+1` otherwise.`
func GetCaller(depth int) string {
	pc, _, _, ok := runtime.Caller(depth + 1)
	if !ok {
		return ""
	}

	funcPath := runtime.FuncForPC(pc).Name()

	// the funcpath base looks something like this:
	// prefix.funcName[...].foo.bar
	// with the [...] only appearing for funcs with generics.
	base := path.Base(funcPath)

	// so when we split it into parts by '.', we get
	// [prefix, funcName[, ], foo, bar]
	parts := strings.Split(base, ".")

	// in certain conditions we'll only get the funcName
	// itself, without the other parts.  In that case, we
	// just need to strip the generic portion from the base.
	if len(parts) < 2 {
		return strings.ReplaceAll(base, "[...]", "")
	}

	// in most cases we'll take the 1th index (the func
	// name) and trim off the bracket that remains from
	// splitting on a period.
	return strings.TrimSuffix(parts[1], "[")
}
