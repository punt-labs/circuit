package circuitrun

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWritePreservesMode(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "check-registry.yaml")
	if err := os.WriteFile(path, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(path, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("mode = %o, want 640", got)
	}
	if err := writeFileAtomically(path, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	current, err := os.Stat(path)
	if err != nil || !os.SameFile(info, current) {
		t.Fatalf("unchanged content replaced the file: %v", err)
	}
}

func TestAtomicWriteUpdatesSymlinkTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "registry.yaml")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "check-registry.yaml")
	if err := os.Symlink(filepath.Base(target), link); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(link, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	destination, err := os.Readlink(link)
	if err != nil || destination != filepath.Base(target) {
		t.Fatalf("symlink changed: %q (%v)", destination, err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "updated" {
		t.Fatalf("target was not updated: %q (%v)", content, err)
	}
}

func TestAtomicWriteCreatesDanglingSymlinkTarget(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	subdir := filepath.Join(dir, "machines")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(subdir, "check-registry.yaml")
	relativeTarget := filepath.Join("..", "registry.yaml")
	if err := os.Symlink(relativeTarget, link); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomically(link, []byte("new registry")); err != nil {
		t.Fatal(err)
	}
	destination, err := os.Readlink(link)
	if err != nil || destination != relativeTarget {
		t.Fatalf("symlink changed: %q (%v)", destination, err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "registry.yaml"))
	if err != nil || string(content) != "new registry" {
		t.Fatalf("target was not created: %q (%v)", content, err)
	}
}
