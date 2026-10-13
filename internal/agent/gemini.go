package agent

import (
	"context"
	"os/exec"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

// Gemini launches Google's `gemini` CLI.
type Gemini struct{}

func (Gemini) ID() string               { return "gemini" }
func (Gemini) DisplayName() string      { return "Gemini CLI" }
func (Gemini) InstructionFile() string  { return "GEMINI.md" }
func (g Gemini) Detect() (string, bool) { return detect(g.ID()) }

// Command runs `gemini [--prompt-interactive prompt]`. A bare positional
// argument would run non-interactively, so the prompt goes through
// --prompt-interactive. Resume is ignored: gemini resumes sessions from
// inside the CLI (/chat resume) rather than via a stable flag.
func (g Gemini) Command(ctx context.Context, p project.Project, o LaunchOptions) *exec.Cmd {
	var args []string
	if o.OpeningPrompt != "" {
		args = append(args, "--prompt-interactive", o.OpeningPrompt)
	}
	return command(ctx, g.ID(), p, args)
}

// Usage is unsupported for gemini in v1.
func (Gemini) Usage(context.Context) (*Usage, error) { return nil, ErrUsageUnsupported }
