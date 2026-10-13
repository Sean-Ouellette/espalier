// Package gitx wraps the git CLI in small typed functions. Every call goes
// through an exec.Runner so tests can script git's output with exec.Fake.
package gitx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	osexec "os/exec"
	"strings"

	"github.com/Sean-Ouellette/espalier/internal/exec"
)

var (
	// ErrGitNotFound means the git binary is not on PATH.
	ErrGitNotFound = errors.New("gitx: git executable not found")
	// ErrNoRemote means the requested remote does not exist in the repo.
	ErrNoRemote = errors.New("gitx: no such remote")
)

// Git runs git commands through a Runner.
type Git struct {
	runner exec.Runner
}

// New returns a Git that executes through r.
func New(r exec.Runner) *Git {
	return &Git{runner: r}
}

// IsRepo reports whether dir is inside a git work tree. A non-zero exit from
// git means "no", not an error; only failures to run git are errors.
func (g *Git) IsRepo(ctx context.Context, dir string) (bool, error) {
	_, _, err := g.run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CurrentBranch returns the checked-out branch name, e.g. "main".
func (g *Git) CurrentBranch(ctx context.Context, dir string) (string, error) {
	out, _, err := g.run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RemoteURL returns the fetch URL for the named remote (usually "origin").
// A remote that does not exist yields ErrNoRemote.
func (g *Git) RemoteURL(ctx context.Context, dir, name string) (string, error) {
	out, _, err := g.run(ctx, dir, "remote", "get-url", name)
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return "", fmt.Errorf("%w: %s", ErrNoRemote, name)
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// run executes git with args in dir. It maps a missing binary to
// ErrGitNotFound and folds stderr into the error for non-zero exits so
// callers (and users) see git's own message. The *exec.ExitError stays in
// the chain so callers can still distinguish "git failed" from "git absent".
func (g *Git) run(ctx context.Context, dir string, args ...string) (stdout, stderr []byte, err error) {
	stdout, stderr, err = g.runner.Run(ctx, dir, "git", args...)
	if err == nil {
		return stdout, stderr, nil
	}
	if errors.Is(err, osexec.ErrNotFound) {
		return stdout, stderr, ErrGitNotFound
	}
	msg := string(bytes.TrimSpace(stderr))
	if msg == "" {
		return stdout, stderr, fmt.Errorf("gitx: git %s: %w", strings.Join(args, " "), err)
	}
	return stdout, stderr, fmt.Errorf("gitx: git %s: %s: %w", strings.Join(args, " "), msg, err)
}
