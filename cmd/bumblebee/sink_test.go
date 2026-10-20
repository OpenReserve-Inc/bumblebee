package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Inventory output carries home-directory paths and MCP server names, so
// a newly created output file must not be readable by other users.
func TestOpenSinkFileCreatesOwnerOnlyFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits do not apply on Windows")
	}
	path := filepath.Join(t.TempDir(), "inventory.ndjson")
	w, closeFn, err := openSink("file", path, false, sinkHTTPOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	if err := closeFn(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got&0o077 != 0 {
		t.Fatalf("output file mode = %o, want no group/other bits", got)
	}
}
