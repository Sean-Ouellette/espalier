# Espalier

Espalier (es-PAL-yay): Go workbench that manages projects and launches the user's own terminal AI
agents (claude, codex, gemini) with GitHub issues/PRs wired in. TUI (Bubble
Tea) + desktop (Wails v3, embedded pty terminal). Full design:
`docs/superpowers/specs/2026-10-06-espalier-design.md`.

## How Claude works in this repo

- **Claude writes the code, feature by feature; the user tests as we go.**
  (Changed 2026-10-13 from the original "user writes all code" learning
  setup.) Each feature: brainstorm/confirm the design, Claude implements with
  tests, user runs it and reviews, then the user commits.
- Explain the Go idioms and the *why* in chat as features land, so the
  project still teaches Go and GitHub integration.
- Claude does not run `gh`/`git` write commands (commit, push, PR) unless
  asked; the user handles those.
- Review feedback should cite `file:line` and name the Go concept involved.

## Layout (see spec §3.1)

- `cmd/espalier` TUI, `cmd/espalier-desktop` Wails app, `internal/` shared core.
- All shell-outs (`git`, `gh`, agents) go through `internal/exec.Runner` so
  tests use the fake runner with fixtures in `testdata/`.
- Per-project data split: `.espalier/` in the repo, app state in
  `~/.config/espalier/`.

## Commands

- `go test ./...`, `go vet ./...`, `golangci-lint run` (CI runs all three).
