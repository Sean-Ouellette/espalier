// Package github wraps the gh CLI. Every call goes through an exec.Runner and
// decodes gh's --json output into Go structs, so tests script gh with
// exec.Fake and fixtures recorded from the real tool under testdata/.
package github

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
	// ErrGHNotInstalled means the gh binary is not on PATH.
	ErrGHNotInstalled = errors.New("github: gh executable not found; install it from https://cli.github.com")
	// ErrNotAuthenticated means gh has no logged-in account; run `gh auth login`.
	ErrNotAuthenticated = errors.New("github: not logged in; run `gh auth login`")
)

// GH runs gh commands through a Runner.
type GH struct {
	runner exec.Runner
}

// New returns a GH that executes through r.
func New(r exec.Runner) *GH {
	return &GH{runner: r}
}

// AuthStatus returns the logged-in GitHub login, or ErrNotAuthenticated.
// It asks the API for the current user rather than parsing the
// human-oriented output of `gh auth status`.
func (g *GH) AuthStatus(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "api", "user", "--jq", ".login")
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		return "", ErrNotAuthenticated
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// run executes gh with args. gh takes the repo via --repo, so no working
// directory is needed. A missing binary maps to ErrGHNotInstalled; a non-zero
// exit folds stderr into the error, keeping *exec.ExitError in the chain.
func (g *GH) run(ctx context.Context, args ...string) ([]byte, error) {
	stdout, stderr, err := g.runner.Run(ctx, "", "gh", args...)
	if err == nil {
		return stdout, nil
	}
	if errors.Is(err, osexec.ErrNotFound) {
		return stdout, ErrGHNotInstalled
	}
	cmd := "gh " + strings.Join(args, " ")
	if msg := string(bytes.TrimSpace(stderr)); msg != "" {
		return stdout, fmt.Errorf("github: %s: %s: %w", cmd, msg, err)
	}
	return stdout, fmt.Errorf("github: %s: %w", cmd, err)
}
