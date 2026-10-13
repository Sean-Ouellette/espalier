package project

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	xexec "github.com/Sean-Ouellette/espalier/internal/exec"
	"github.com/Sean-Ouellette/espalier/internal/gitx"
)

func exitErr(t *testing.T) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit 128").Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	return ee
}

func TestOpenGitHubRepo(t *testing.T) {
	dir := t.TempDir()
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Stdout: []byte("true\n")},
		xexec.Key("git", "remote", "get-url", "origin"):        {Stdout: []byte("git@github.com:Sean-Ouellette/espalier.git\n")},
	}}

	p, err := Open(context.Background(), gitx.New(fake), dir)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if p.Path != dir {
		t.Errorf("Path = %q, want %q", p.Path, dir)
	}
	if p.Name != filepath.Base(dir) {
		t.Errorf("Name = %q, want %q", p.Name, filepath.Base(dir))
	}
	if p.Remote == nil || *p.Remote != (Remote{Owner: "Sean-Ouellette", Repo: "espalier"}) {
		t.Errorf("Remote = %+v, want Sean-Ouellette/espalier", p.Remote)
	}
	if len(p.ID) != 8 {
		t.Errorf("ID = %q, want 8 chars", p.ID)
	}
	if p.AddedAt.IsZero() {
		t.Error("AddedAt is zero")
	}
	for _, c := range fake.Calls {
		if c.Dir != dir {
			t.Errorf("git ran in %q, want %q", c.Dir, dir)
		}
	}
}

func TestOpenRepoWithoutRemoteHasNilRemote(t *testing.T) {
	dir := t.TempDir()
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Stdout: []byte("true\n")},
		xexec.Key("git", "remote", "get-url", "origin"): {
			Stderr: []byte("error: No such remote 'origin'\n"), Err: exitErr(t),
		},
	}}

	p, err := Open(context.Background(), gitx.New(fake), dir)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if p.Remote != nil {
		t.Errorf("Remote = %+v, want nil", p.Remote)
	}
}

func TestOpenNonGitHubRemoteHasNilRemote(t *testing.T) {
	dir := t.TempDir()
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Stdout: []byte("true\n")},
		xexec.Key("git", "remote", "get-url", "origin"):        {Stdout: []byte("https://gitlab.com/x/y.git\n")},
	}}

	p, err := Open(context.Background(), gitx.New(fake), dir)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if p.Remote != nil {
		t.Errorf("Remote = %+v, want nil", p.Remote)
	}
}

func TestOpenNotARepo(t *testing.T) {
	dir := t.TempDir()
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {
			Stderr: []byte("fatal: not a git repository\n"), Err: exitErr(t),
		},
	}}

	_, err := Open(context.Background(), gitx.New(fake), dir)
	if !errors.Is(err, ErrNotRepo) {
		t.Errorf("Open() error = %v, want ErrNotRepo", err)
	}
}

func TestOpenMissingDirectory(t *testing.T) {
	fake := &xexec.Fake{}

	_, err := Open(context.Background(), gitx.New(fake), filepath.Join(t.TempDir(), "nope"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Open() error = %v, want os.ErrNotExist", err)
	}
	if len(fake.Calls) != 0 {
		t.Errorf("git was called %d times for a missing dir, want 0", len(fake.Calls))
	}
}

func TestOpenResolvesRelativePath(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Stdout: []byte("true\n")},
		xexec.Key("git", "remote", "get-url", "origin"): {
			Stderr: []byte("error: No such remote 'origin'\n"), Err: exitErr(t),
		},
	}}

	p, err := Open(context.Background(), gitx.New(fake), ".")
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if !filepath.IsAbs(p.Path) {
		t.Errorf("Path = %q, want absolute", p.Path)
	}
}
