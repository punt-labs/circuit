package circuitrun

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomically(path string, content []byte) error {
	path, existing, err := atomicWriteTarget(path)
	if err != nil {
		return err
	}
	if existing != nil {
		previous, err := os.ReadFile(path) //nolint:gosec // G304: resolved path for the file being updated
		if err == nil && bytes.Equal(previous, content) {
			return nil
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".circuit-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer func() {
		_ = os.Remove(temp.Name()) //nolint:errcheck // Best-effort cleanup; the file may already have been renamed.
	}()
	if err := errors.Join(writeAndSyncFile(temp, content, existing), temp.Close()); err != nil {
		return fmt.Errorf("flush temporary file: %w", err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	return nil
}

func atomicWriteTarget(path string) (string, os.FileInfo, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("inspect destination: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := atomicSymlinkTarget(path)
		if err != nil {
			return "", nil, fmt.Errorf("resolve destination: %w", err)
		}
		return atomicWriteTarget(target)
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("destination is not a regular file: %s", path)
	}
	return path, info, nil
}

func atomicSymlinkTarget(path string) (string, error) {
	target, err := filepath.EvalSymlinks(path)
	if err == nil {
		return target, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("resolve symlink: %w", err)
	}
	target, err = os.Readlink(path)
	if err != nil {
		return "", fmt.Errorf("read dangling symlink: %w", err)
	}
	if !filepath.IsAbs(target) {
		linkDir, _ := filepath.Split(path)
		target = linkDir + target
	}
	parent, name := filepath.Split(target)
	if name == "" {
		return "", fmt.Errorf("symlink target has no file name: %s", target)
	}
	if parent == "" {
		parent = "."
	}
	dir, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", fmt.Errorf("resolve symlink directory: %w", err)
	}
	return filepath.Join(dir, name), nil
}

func writeAndSyncFile(file *os.File, content []byte, existing os.FileInfo) error {
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("write file data: %w", err)
	}
	if existing != nil {
		mode := existing.Mode() & (os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		if err := file.Chmod(mode); err != nil {
			return fmt.Errorf("preserve file mode: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync file data: %w", err)
	}
	return nil
}
