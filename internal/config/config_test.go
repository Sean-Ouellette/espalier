package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDirHonorsEnvOverride(t *testing.T) {
	t.Setenv("ESPALIER_CONFIG_DIR", "/tmp/override")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if got != "/tmp/override" {
		t.Errorf("Dir() = %q, want %q", got, "/tmp/override")
	}
}

func TestDirDefaultsToUserConfigDir(t *testing.T) {
	t.Setenv("ESPALIER_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if want := filepath.Join("/tmp/xdg", "espalier"); got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !got.Equal(Default()) {
		t.Errorf("Load() = %+v, want defaults %+v", got, Default())
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "espalier")
	want := Settings{DefaultAgent: "codex", UsageWindows: []string{"5h", "7d"}}

	if err := Save(dir, want); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !got.Equal(want) {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestSaveLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()

	if err := Save(dir, Default()); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "settings.json" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("dir contains %v, want only settings.json", names)
	}
}

func TestLoadCorruptFileBacksUpAndReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(dir)

	var ce *CorruptError
	if !errors.As(err, &ce) {
		t.Fatalf("Load() error = %v, want *CorruptError", err)
	}
	if ce.Backup != path+".bak" {
		t.Errorf("CorruptError.Backup = %q, want %q", ce.Backup, path+".bak")
	}
	if !got.Equal(Default()) {
		t.Errorf("Load() = %+v, want defaults %+v", got, Default())
	}
	if _, statErr := os.Stat(path + ".bak"); statErr != nil {
		t.Errorf("backup file missing: %v", statErr)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("corrupt file still present (stat err = %v), want removed", statErr)
	}
}
