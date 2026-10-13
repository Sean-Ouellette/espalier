// Package config resolves Espalier's config directory and loads/saves
// settings.json with defaults and atomic writes.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// EnvDir is the environment variable that overrides the config directory.
const EnvDir = "ESPALIER_CONFIG_DIR"

const settingsFile = "settings.json"

// Settings is the app-wide user configuration stored in settings.json.
type Settings struct {
	// DefaultAgent is the agent ID used when a project has no override.
	DefaultAgent string `json:"default_agent"`
	// UsageWindows lists which usage windows to display, e.g. "5h", "7d".
	UsageWindows []string `json:"usage_windows"`
}

// Default returns the settings used when no file exists.
func Default() Settings {
	return Settings{
		DefaultAgent: "claude",
		UsageWindows: []string{"5h", "7d"},
	}
}

// Equal reports whether two Settings have the same values.
// Settings contains a slice, so it cannot be compared with ==.
func (s Settings) Equal(o Settings) bool {
	return s.DefaultAgent == o.DefaultAgent &&
		slices.Equal(s.UsageWindows, o.UsageWindows)
}

// CorruptError reports that settings.json could not be parsed. The bad file
// has been moved to Backup and defaults were returned in its place.
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

// Dir returns the config directory: $ESPALIER_CONFIG_DIR if set, otherwise
// $XDG_CONFIG_HOME/espalier (defaulting to ~/.config/espalier).
func Dir() (string, error) {
	if d := os.Getenv(EnvDir); d != "" {
		return d, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: resolve config dir: %w", err)
	}
	return filepath.Join(base, "espalier"), nil
}

// Load reads settings.json from dir. A missing file yields Default() with a
// nil error. An unparsable file is renamed to settings.json.bak and Default()
// is returned together with a *CorruptError so the UI can tell the user.
func Load(dir string) (Settings, error) {
	path := filepath.Join(dir, settingsFile)

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("config: read %s: %w", path, err)
	}

	s := Default()
	if err := json.Unmarshal(data, &s); err != nil {
		backup := path + ".bak"
		if renameErr := os.Rename(path, backup); renameErr != nil {
			return Default(), fmt.Errorf("config: back up corrupt %s: %w", path, renameErr)
		}
		return Default(), &CorruptError{Path: path, Backup: backup, Err: err}
	}
	return s, nil
}

// Save writes s to settings.json in dir atomically: it creates dir if needed,
// writes a temp file beside the target, fsyncs it, then renames it into place
// so readers never observe a half-written file.
func Save(dir string, s Settings) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("config: create %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode settings: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, settingsFile+".*.tmp")
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

	target := filepath.Join(dir, settingsFile)
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("config: replace %s: %w", target, err)
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
