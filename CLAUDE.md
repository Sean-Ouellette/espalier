# Espalier

Espalier (es-PAL-yay): Go workbench that manages projects and launches the user's own terminal AI
agents (claude, codex, gemini) with GitHub issues/PRs wired in. TUI (Bubble
Tea) + desktop (Wails v3, embedded pty terminal). Full design:
`docs/superpowers/specs/2026-10-06-espalier-design.md`.

## How Claude works in this repo

- **The user writes all code.** This is a learning project for Go and GitHub
  integration. Claude explains, points to docs, reviews, and helps debug. Do
  not write implementation files, scaffold packages, or run `gh`/`git` write
  commands on the user's behalf. Short illustrative snippets in chat are fine.
- Prefer teaching the Go idiom and the *why* over giving the answer.
- Review feedback should cite `file:line` and name the Go concept involved.

## Layout (see spec §3.1)

- `cmd/espalier` TUI, `cmd/espalier-desktop` Wails app, `internal/` shared core.
- All shell-outs (`git`, `gh`, agents) go through `internal/exec.Runner` so
  tests use the fake runner with fixtures in `testdata/`.
- Per-project data split: `.espalier/` in the repo, app state in
  `~/.config/espalier/`.

## Commands

- `go test ./...`, `go vet ./...`, `golangci-lint run` (CI runs all three).
