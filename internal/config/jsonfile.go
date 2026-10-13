package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// CorruptError reports that a JSON file could not be parsed. The bad file
// has been moved to Backup and the caller fell back to defaults.
type CorruptError struct {
	Path   string
	Backup string
	Err    error
}

func (e *CorruptError) Error() string {
	return fmt.Sprintf("config: %s is corrupt (moved to %s): %v", e.Path, e.Backup, e.Err)
}

// Unwrap lets errors.Is/As see the underlying parse error.
func (e *CorruptError) Unwrap() error { return e.Err }

// ReadJSON decodes the file at path into v. A missing file is not an error
// and leaves v untouched, so callers pre-fill v with defaults. An unparsable
// file is renamed to path+".bak" and reported as a *CorruptError, again
// leaving v untouched.
func ReadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		backup := path + ".bak"
		if renameErr := os.Rename(path, backup); renameErr != nil {
			return fmt.Errorf("config: back up corrupt %s: %w", path, renameErr)
		}
		return &CorruptError{Path: path, Backup: backup, Err: err}
	}
	return nil
}

// WriteJSON encodes v as indented JSON and writes it atomically to path.
func WriteJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode %s: %w", path, err)
	}
	return WriteFileAtomic(path, append(data, '\n'))
}
