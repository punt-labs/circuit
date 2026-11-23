package circuitrun

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func writeFileAtomically(path string, content []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".circuit-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	defer func() {
		_ = os.Remove(temp.Name()) //nolint:errcheck // Best-effort cleanup; the file may already have been renamed.
	}()
	_, err = temp.Write(content)
	if err == nil {
		err = temp.Sync()
	}
	if err := errors.Join(err, temp.Close()); err != nil {
		return fmt.Errorf("flush temporary file: %w", err)
	}
	if err := os.Rename(temp.Name(), path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	return nil
}
