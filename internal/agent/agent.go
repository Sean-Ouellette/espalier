// Package agent defines the Agent interface Espalier uses to launch the
// user's own terminal AI CLIs, plus one adapter per supported tool.
// Adding an agent is one file implementing Agent and a Registry entry.
package agent

import (
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

// ErrUsageUnsupported means the adapter has no usage source.
var ErrUsageUnsupported = errors.New("agent: usage reporting not supported")

// lookPath is exec.LookPath behind a variable so tests can fake which
// binaries are installed without touching PATH.
var lookPath = exec.LookPath

// Agent is one terminal AI CLI.
type Agent interface {
	ID() string          // "claude" | "codex" | "gemini"; also the binary name
	DisplayName() string // human-readable
	// Detect reports where the binary is on PATH, if at all.
	Detect() (path string, ok bool)
	// InstructionFile is the project-level instructions file the agent reads.
	InstructionFile() string
	// Command builds the interactive process to run in p.Path. It is not
	// started; the caller wires stdin/stdout (a tty or a pty) and runs it.
	Command(ctx context.Context, p project.Project, o LaunchOptions) *exec.Cmd
	// Usage reports subscription usage, or ErrUsageUnsupported.
	Usage(ctx context.Context) (*Usage, error)
}

// LaunchOptions tunes a launch.
type LaunchOptions struct {
	OpeningPrompt string // e.g. "Work on issue #12: …" or the bootstrap prompt
	Resume        bool   // continue the most recent session instead of a new one
}

// Usage is a snapshot of an agent's rate-limit windows.
type Usage struct {
	Windows   []UsageWindow
	FetchedAt time.Time
}

// UsageWindow is one rate-limit window, e.g. five-hour or seven-day.
type UsageWindow struct {
	Kind     string // stable key used in settings to show/hide
	Label    string
	Percent  float64
	ResetsAt time.Time
}

// detect is the shared Detect implementation.
func detect(binary string) (string, bool) {
	path, err := lookPath(binary)
	if err != nil {
		return "", false
	}
	return path, true
}

// command builds an exec.Cmd for binary with args, running in p.Path.
func command(ctx context.Context, binary string, p project.Project, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = p.Path
	return cmd
}
