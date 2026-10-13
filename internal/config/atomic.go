package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path so readers never see a partial file:
// it creates the parent directory (0700) if needed, writes a temp file
// beside the target, fsyncs it, sets mode 0600, then renames it into place.
// Rename within one directory is atomic on POSIX filesystems.
func WriteFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// On any failure below, remove the temp file so the dir stays clean.
	// After a successful rename the file is gone and Remove is a no-op.
	defer func() { _ = os.Remove(tmpName) }()

	if err := writeAndClose(tmp, data); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("config: chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: replace %s: %w", path, err)
	}
	return nil
}

// writeAndClose writes data, fsyncs, and closes f, always closing even when
// an earlier step fails so the descriptor is never leaked.
func writeAndClose(f *os.File, data []byte) (err error) {
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("config: close temp file: %w", cerr)
		}
	}()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("config: sync temp file: %w", err)
	}
	return nil
}
