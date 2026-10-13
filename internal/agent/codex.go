package agent

import (
	"context"
	"os/exec"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

// Codex launches OpenAI's `codex` CLI.
type Codex struct{}

func (Codex) ID() string               { return "codex" }
func (Codex) DisplayName() string      { return "Codex CLI" }
func (Codex) InstructionFile() string  { return "AGENTS.md" }
func (c Codex) Detect() (string, bool) { return detect(c.ID()) }

// Command runs `codex [prompt]` or, to resume, `codex resume --last [prompt]`.
// Flags follow the codex docs; codex is not installed on the dev machine, so
// verify against `codex --help` when it is.
func (c Codex) Command(ctx context.Context, p project.Project, o LaunchOptions) *exec.Cmd {
	var args []string
	if o.Resume {
		args = append(args, "resume", "--last")
	}
	if o.OpeningPrompt != "" {
		args = append(args, o.OpeningPrompt)
	}
	return command(ctx, c.ID(), p, args)
}

// Usage is unsupported for codex in v1.
func (Codex) Usage(context.Context) (*Usage, error) { return nil, ErrUsageUnsupported }
