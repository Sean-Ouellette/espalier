package agent

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

var proj = project.Project{ID: "abcd1234", Name: "espalier", Path: "/home/x/espalier"}

// fakeLookPath makes only the named binaries "installed" for one test.
func fakeLookPath(t *testing.T, installed ...string) {
	t.Helper()
	orig := lookPath
	lookPath = func(file string) (string, error) {
		for _, name := range installed {
			if name == file {
				return "/usr/bin/" + file, nil
			}
		}
		return "", errors.New("not found")
	}
	t.Cleanup(func() { lookPath = orig })
}

func TestAdapterIdentity(t *testing.T) {
	tests := []struct {
		agent       Agent
		id, name    string
		instruction string
	}{
		{Claude{}, "claude", "Claude Code", "CLAUDE.md"},
		{Codex{}, "codex", "Codex CLI", "AGENTS.md"},
		{Gemini{}, "gemini", "Gemini CLI", "GEMINI.md"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := tt.agent.ID(); got != tt.id {
				t.Errorf("ID() = %q, want %q", got, tt.id)
			}
			if got := tt.agent.DisplayName(); got != tt.name {
				t.Errorf("DisplayName() = %q, want %q", got, tt.name)
			}
			if got := tt.agent.InstructionFile(); got != tt.instruction {
				t.Errorf("InstructionFile() = %q, want %q", got, tt.instruction)
			}
		})
	}
}

func TestDetect(t *testing.T) {
	fakeLookPath(t, "claude", "gemini")

	if path, ok := (Claude{}).Detect(); !ok || path != "/usr/bin/claude" {
		t.Errorf("Claude.Detect() = %q, %v; want /usr/bin/claude, true", path, ok)
	}
	if path, ok := (Codex{}).Detect(); ok || path != "" {
		t.Errorf("Codex.Detect() = %q, %v; want \"\", false", path, ok)
	}
}

func TestCommandArgs(t *testing.T) {
	tests := []struct {
		name  string
		agent Agent
		opts  LaunchOptions
		want  []string // argv after the binary name
	}{
		{"claude plain", Claude{}, LaunchOptions{}, nil},
		{"claude prompt", Claude{}, LaunchOptions{OpeningPrompt: "Work on issue #4"}, []string{"Work on issue #4"}},
		{"claude resume", Claude{}, LaunchOptions{Resume: true}, []string{"--continue"}},
		{"claude resume+prompt", Claude{}, LaunchOptions{Resume: true, OpeningPrompt: "hi"}, []string{"--continue", "hi"}},

		{"codex plain", Codex{}, LaunchOptions{}, nil},
		{"codex prompt", Codex{}, LaunchOptions{OpeningPrompt: "hi"}, []string{"hi"}},
		{"codex resume", Codex{}, LaunchOptions{Resume: true}, []string{"resume", "--last"}},
		{"codex resume+prompt", Codex{}, LaunchOptions{Resume: true, OpeningPrompt: "hi"}, []string{"resume", "--last", "hi"}},

		{"gemini plain", Gemini{}, LaunchOptions{}, nil},
		{"gemini prompt", Gemini{}, LaunchOptions{OpeningPrompt: "hi"}, []string{"--prompt-interactive", "hi"}},
		{"gemini resume is a no-op", Gemini{}, LaunchOptions{Resume: true, OpeningPrompt: "hi"}, []string{"--prompt-interactive", "hi"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.agent.Command(context.Background(), proj, tt.opts)
			if cmd.Dir != proj.Path {
				t.Errorf("Dir = %q, want %q", cmd.Dir, proj.Path)
			}
			// cmd.Args[0] is the binary; compare the rest.
			// slices.Equal treats a nil and an empty slice as equal, unlike
			// reflect.DeepEqual, which is why it fits the "plain" cases.
			if got := cmd.Args[1:]; !slices.Equal(got, tt.want) {
				t.Errorf("args = %q, want %q", got, tt.want)
			}
			if cmd.Args[0] != tt.agent.ID() {
				t.Errorf("Args[0] = %q, want binary name %q", cmd.Args[0], tt.agent.ID())
			}
		})
	}
}

func TestUsageUnsupportedForAll(t *testing.T) {
	for _, a := range []Agent{Claude{}, Codex{}, Gemini{}} {
		_, err := a.Usage(context.Background())
		if !errors.Is(err, ErrUsageUnsupported) {
			t.Errorf("%s.Usage() error = %v, want ErrUsageUnsupported", a.ID(), err)
		}
	}
}

func TestRegistryAllOrderAndGet(t *testing.T) {
	reg := NewRegistry()

	var ids []string
	for _, a := range reg.All() {
		ids = append(ids, a.ID())
	}
	if want := []string{"claude", "codex", "gemini"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("All() ids = %v, want %v", ids, want)
	}

	if a, ok := reg.Get("codex"); !ok || a.ID() != "codex" {
		t.Errorf("Get(codex) = %v, %v", a, ok)
	}
	if _, ok := reg.Get("nope"); ok {
		t.Error("Get(nope) ok = true, want false")
	}
}

func TestRegistryInstalled(t *testing.T) {
	fakeLookPath(t, "gemini")
	reg := NewRegistry()

	got := reg.Installed()
	if len(got) != 1 || got[0].ID() != "gemini" {
		t.Errorf("Installed() = %v, want [gemini]", got)
	}
}
