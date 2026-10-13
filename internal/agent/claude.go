package agent

import (
	"context"
	"os/exec"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

// Claude launches Anthropic's `claude` CLI.
type Claude struct{}

func (Claude) ID() string               { return "claude" }
func (Claude) DisplayName() string      { return "Claude Code" }
func (Claude) InstructionFile() string  { return "CLAUDE.md" }
func (c Claude) Detect() (string, bool) { return detect(c.ID()) }

// Command runs `claude [--continue] [prompt]`. The prompt is a positional
// argument; --continue resumes the most recent conversation in this directory.
func (c Claude) Command(ctx context.Context, p project.Project, o LaunchOptions) *exec.Cmd {
	var args []string
	if o.Resume {
		args = append(args, "--continue")
	}
	if o.OpeningPrompt != "" {
		args = append(args, o.OpeningPrompt)
	}
	return command(ctx, c.ID(), p, args)
}

// Usage is not wired yet; the OAuth usage endpoint lands in a later slice.
func (Claude) Usage(context.Context) (*Usage, error) { return nil, ErrUsageUnsupported }
