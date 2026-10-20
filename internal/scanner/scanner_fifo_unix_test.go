//go:build unix

package scanner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/perplexityai/bumblebee/internal/model"
	"github.com/perplexityai/bumblebee/internal/output"
)

// A FIFO named like a lockfile, directly or behind a symlink, must be
// skipped rather than opened: open(2) on a FIFO blocks until a writer
// appears, which would hang the scan.
func TestFIFONamedLikeLockfileDoesNotHang(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "package-lock.json"), 0o600); err != nil {
		t.Skipf("mkfifo not supported: %v", err)
	}
	linked := filepath.Join(root, "linked")
	if err := os.MkdirAll(linked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "package-lock.json"), filepath.Join(linked, "package-lock.json")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	// A symlink to a regular lockfile must still be scanned.
	real := filepath.Join(root, "real", "package-lock.json")
	writeFile(t, real, `{"lockfileVersion":3,"packages":{"node_modules/foo":{"version":"1.0.0"}}}`)
	viaLink := filepath.Join(root, "vialink")
	if err := os.MkdirAll(viaLink, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(viaLink, "package-lock.json")); err != nil {
		t.Fatal(err)
	}

	em := output.New(&bytes.Buffer{}, &bytes.Buffer{}, "r")
	done := make(chan error, 1)
	go func() {
		_, err := Run(context.Background(), Config{
			Roots:       []Root{{Path: root, Kind: model.RootKindProject}},
			Profile:     model.ProfileProject,
			MaxFileSize: 1 << 20,
			Concurrency: 2,
			Emitter:     em,
		})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("scan: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("scan did not terminate; FIFO was opened")
	}
	if em.RecordsEmitted < 1 {
		t.Errorf("expected the symlinked regular lockfile to be scanned, got %d records", em.RecordsEmitted)
	}
}
