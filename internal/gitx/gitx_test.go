package gitx

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	xexec "github.com/Sean-Ouellette/espalier/internal/exec"
)

// exitErr builds a non-zero exit error the way os/exec reports one, without
// running a process: Go lets us construct ExitError around a ProcessState.
func exitErr(t *testing.T) error {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit 128")
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	return ee
}

func TestIsRepoTrueInsideWorkTree(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Stdout: []byte("true\n")},
	}}
	g := New(fake)

	ok, err := g.IsRepo(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("IsRepo() error: %v", err)
	}
	if !ok {
		t.Error("IsRepo() = false, want true")
	}
	want := []xexec.Call{{Dir: "/repo", Name: "git", Args: []string{"rev-parse", "--is-inside-work-tree"}}}
	if !reflect.DeepEqual(fake.Calls, want) {
		t.Errorf("calls = %+v, want %+v", fake.Calls, want)
	}
}

func TestIsRepoFalseOnNonZeroExit(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {
			Stderr: []byte("fatal: not a git repository\n"),
			Err:    exitErr(t),
		},
	}}
	g := New(fake)

	ok, err := g.IsRepo(context.Background(), "/not-repo")
	if err != nil {
		t.Fatalf("IsRepo() error: %v, want nil for non-zero exit", err)
	}
	if ok {
		t.Error("IsRepo() = true, want false")
	}
}

func TestIsRepoGitMissingReturnsSentinel(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--is-inside-work-tree"): {Err: exec.ErrNotFound},
	}}
	g := New(fake)

	_, err := g.IsRepo(context.Background(), "/repo")
	if !errors.Is(err, ErrGitNotFound) {
		t.Errorf("IsRepo() error = %v, want ErrGitNotFound", err)
	}
}

func TestCurrentBranchTrimsOutput(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--abbrev-ref", "HEAD"): {Stdout: []byte("main\n")},
	}}
	g := New(fake)

	got, err := g.CurrentBranch(context.Background(), "/repo")
	if err != nil {
		t.Fatalf("CurrentBranch() error: %v", err)
	}
	if got != "main" {
		t.Errorf("CurrentBranch() = %q, want %q", got, "main")
	}
}

func TestCurrentBranchFailureWrapsStderr(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "rev-parse", "--abbrev-ref", "HEAD"): {
			Stderr: []byte("fatal: not a git repository\n"),
			Err:    exitErr(t),
		},
	}}
	g := New(fake)

	_, err := g.CurrentBranch(context.Background(), "/repo")
	if err == nil {
		t.Fatal("CurrentBranch() error = nil, want failure")
	}
	if got := err.Error(); !strings.Contains(got, "not a git repository") {
		t.Errorf("error %q should include git's stderr", got)
	}
}

func TestRemoteURLReturnsTrimmedURL(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "remote", "get-url", "origin"): {Stdout: []byte("git@github.com:Sean-Ouellette/espalier.git\n")},
	}}
	g := New(fake)

	got, err := g.RemoteURL(context.Background(), "/repo", "origin")
	if err != nil {
		t.Fatalf("RemoteURL() error: %v", err)
	}
	if want := "git@github.com:Sean-Ouellette/espalier.git"; got != want {
		t.Errorf("RemoteURL() = %q, want %q", got, want)
	}
}

func TestRemoteURLMissingRemoteReturnsSentinel(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("git", "remote", "get-url", "origin"): {
			Stderr: []byte("error: No such remote 'origin'\n"),
			Err:    exitErr(t),
		},
	}}
	g := New(fake)

	_, err := g.RemoteURL(context.Background(), "/repo", "origin")
	if !errors.Is(err, ErrNoRemote) {
		t.Errorf("RemoteURL() error = %v, want ErrNoRemote", err)
	}
}
