package exec

import (
	"bytes"
	"context"
	osexec "os/exec"
)

// Runner executes an external command in dir and returns its stdout, stderr, and exit error.
// Tests substitute a Fake; production uses Real
type Runner interface {
	Run(ctx context.Context, dir string, name string, args ...string) (stdout, stderr []byte, err error)
}

// Real is a Runner backed by os/exec
type Real struct{}

// Run implements Runner
func (Real) Run(ctx context.Context, dir string, name string, args ...string) ([]byte, []byte, error) {
	cmd := osexec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
