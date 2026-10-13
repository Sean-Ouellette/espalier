// Package config resolves Espalier's config directory and loads/saves
// settings.json with defaults and atomic writes.
package config

import (
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
// Decoding into a Default() value means keys missing from an older file keep
// their default rather than becoming zero values.
func Load(dir string) (Settings, error) {
	s := Default()
	if err := ReadJSON(filepath.Join(dir, settingsFile), &s); err != nil {
		return Default(), err
	}
	return s, nil
}

// Save writes s to settings.json in dir atomically (see WriteFileAtomic).
func Save(dir string, s Settings) error {
	return WriteJSON(filepath.Join(dir, settingsFile), s)
}
