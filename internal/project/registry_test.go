package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func sample(id, name string) Project {
	return Project{
		ID: id, Name: name, Path: "/home/x/" + name,
		Remote:  &Remote{Owner: "o", Repo: name},
		AddedAt: time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC),
	}
}

func TestLoadRegistryMissingFileIsEmpty(t *testing.T) {
	reg, err := LoadRegistry(t.TempDir())
	if err != nil {
		t.Fatalf("LoadRegistry() error: %v", err)
	}
	if got := reg.List(); len(got) != 0 {
		t.Errorf("List() = %v, want empty", got)
	}
}

func TestRegistryAddGetListRemove(t *testing.T) {
	reg, _ := LoadRegistry(t.TempDir())
	b, a := sample("bbbbbbbb", "beta"), sample("aaaaaaaa", "alpha")

	if err := reg.Add(b); err != nil {
		t.Fatalf("Add(b) error: %v", err)
	}
	if err := reg.Add(a); err != nil {
		t.Fatalf("Add(a) error: %v", err)
	}

	got, ok := reg.Get("aaaaaaaa")
	if !ok || got.Name != "alpha" {
		t.Errorf("Get(a) = %+v, %v; want alpha, true", got, ok)
	}
	if _, ok := reg.Get("missing"); ok {
		t.Error("Get(missing) ok = true, want false")
	}

	list := reg.List()
	if len(list) != 2 || list[0].Name != "alpha" || list[1].Name != "beta" {
		t.Errorf("List() order = %v, want [alpha beta]", names(list))
	}

	if !reg.Remove("bbbbbbbb") {
		t.Error("Remove(b) = false, want true")
	}
	if reg.Remove("bbbbbbbb") {
		t.Error("Remove(b) again = true, want false")
	}
	if list := reg.List(); len(list) != 1 || list[0].Name != "alpha" {
		t.Errorf("List() after remove = %v, want [alpha]", names(list))
	}
}

func TestRegistryAddRejectsDuplicatePath(t *testing.T) {
	reg, _ := LoadRegistry(t.TempDir())
	p := sample("aaaaaaaa", "alpha")
	if err := reg.Add(p); err != nil {
		t.Fatal(err)
	}

	dup := p
	dup.ID = "cccccccc"
	err := reg.Add(dup)
	if !errors.Is(err, ErrAlreadyAdded) {
		t.Errorf("Add(dup) error = %v, want ErrAlreadyAdded", err)
	}
	if len(reg.List()) != 1 {
		t.Errorf("registry has %d projects, want 1", len(reg.List()))
	}
}

func TestRegistrySaveThenLoadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	reg, _ := LoadRegistry(dir)
	a := sample("aaaaaaaa", "alpha")
	noRemote := sample("bbbbbbbb", "beta")
	noRemote.Remote = nil
	noRemote.Agent = "codex"
	_ = reg.Add(a)
	_ = reg.Add(noRemote)

	if err := reg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "projects.json")); err != nil {
		t.Fatalf("projects.json not written: %v", err)
	}

	again, err := LoadRegistry(dir)
	if err != nil {
		t.Fatalf("LoadRegistry() error: %v", err)
	}
	got := again.List()
	if len(got) != 2 {
		t.Fatalf("List() has %d projects, want 2", len(got))
	}
	if got[0].Remote == nil || *got[0].Remote != *a.Remote || !got[0].AddedAt.Equal(a.AddedAt) {
		t.Errorf("alpha = %+v, want %+v", got[0], a)
	}
	if got[1].Remote != nil || got[1].Agent != "codex" {
		t.Errorf("beta = %+v, want nil remote and agent codex", got[1])
	}
}

func TestLoadRegistryCorruptFileBacksUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.json")
	if err := os.WriteFile(path, []byte("[oops"), 0o600); err != nil {
		t.Fatal(err)
	}

	reg, err := LoadRegistry(dir)
	var ce *CorruptError
	if !errors.As(err, &ce) {
		t.Fatalf("LoadRegistry() error = %v, want *CorruptError", err)
	}
	if ce.Backup != path+".bak" {
		t.Errorf("Backup = %q, want %q", ce.Backup, path+".bak")
	}
	if reg == nil || len(reg.List()) != 0 {
		t.Errorf("registry after corrupt load = %v, want empty usable registry", reg)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Errorf("backup missing: %v", err)
	}
}

func names(ps []Project) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.Name
	}
	return out
}
