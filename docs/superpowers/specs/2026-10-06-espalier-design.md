# Espalier — Design Spec

**Date:** 2026-10-06
**Status:** Draft for review
**Author:** Sean Ouellette (with Claude as design partner)

---

## 1. Purpose

**Espalier** (es-PAL-yay): the centuries-old craft of training a tree along a frame so it grows where you want. Here, the agent is the tree and the app is the frame.

A personal developer workbench that manages software projects and launches the
user's own terminal AI agents (Claude Code, OpenAI Codex CLI, Gemini CLI, …)
inside them, with GitHub issues and pull requests wired in.

It exists for two reasons:

1. **To make day-to-day development with AI agents smoother**: pick a project,
   see its issues, start the right agent with the right instructions, review
   what it changed, feed review comments back to it.
2. **To be a learning project.** The author writes every line. The goals are
   to learn Go deeply and to learn how GitHub integration works end to end.

### Non-goals (for now)

- Not a chat client. The app never calls an LLM API itself. It launches the
  user's installed agent CLIs so each vendor's own login and limits apply.
- Not a replacement for the agent's own UI. The agent runs in a real terminal.
- Not multi-user, not hosted, not a cloud service.
- Not cross-platform in v1. Linux first. Nothing in the design blocks macOS
  or Windows later, but they are not tested.

---

## 2. Decisions already made

| Decision | Choice | Why |
|---|---|---|
| Language | Go for everything | Learning goal; single toolchain |
| LLM access | **Launcher model**: run the user's own installed agent CLIs | Uses existing subscriptions; avoids Anthropic's third-party-login terms; adding a new LLM is a small adapter |
| TUI | Bubble Tea + Lip Gloss + Bubbles | Standard Go TUI stack |
| Desktop | Wails v3 (beta, API stable). Fall back to v2 if v3 blocks | Native window, web UI, `.desktop` entry so it pins to the taskbar |
| Terminal in desktop | Embedded pane: Go pty (`creack/pty`) ↔ xterm.js | One window; still the user's own CLI |
| Diff / PR review | Desktop only | Hard to render well in a terminal |
| TUI scope | Lighter companion: projects, issues, launch, usage | Keeps the TUI buildable; desktop is the full product |
| GitHub access | Shell out to `gh` | Already authenticated; learn the GitHub model through its CLI first; swap for the REST API later if wanted |
| Git access | Shell out to `git` | Same reasoning; avoids a heavy pure-Go git dependency |
| Per-project data | **Split**: intent + generated instructions in the repo under `.espalier/` and the agent's own file; app state in `~/.config/espalier/` | Instructions travel with the project; app-only state stays out of other people's clones |
| Repo structure | One Go module, two binaries (`cmd/espalier`, `cmd/espalier-desktop`), shared `internal/` core | Standard Go layout; one `go test ./...`; daemon split possible later |

---

## 3. Architecture

```
┌──────────────────────────┐   ┌────────────────────────────────────┐
│  cmd/espalier (TUI)       │   │  cmd/espalier-desktop (Wails v3)    │
│  Bubble Tea              │   │  Go backend  ⇄  web frontend       │
│  - project list          │   │  - everything the TUI has          │
│  - issues                │   │  - setup wizard                    │
│  - launch agent (fg)     │   │  - embedded terminal (pty↔xterm)   │
│  - usage                 │   │  - diff / PR review → send to agent│
└───────────┬──────────────┘   └───────────────┬────────────────────┘
            │                                  │
            └──────────────┬───────────────────┘
                           ▼
            ┌──────────────────────────────┐
            │  internal/ (core, no UI)     │
            │  config    project   gitx    │
            │  github    agent     intent  │
            │  term      exec                │
            └──────────────┬───────────────┘
                           ▼
          git · gh · claude · codex · gemini · (pty)
```

Both binaries import the core directly and run in-process. The core has no
knowledge of either UI.

### 3.1 Repository layout

```
espalier/
├── go.mod                        module github.com/Sean-Ouellette/espalier
├── cmd/
│   ├── espalier/                 TUI entry point
│   └── espalier-desktop/          Wails app: main.go, services, frontend/
├── internal/
│   ├── exec/        Runner interface over os/exec; fake for tests
│   ├── config/      XDG paths, settings.json load/save
│   ├── project/     Project type, registry (projects.json), open/create
│   ├── gitx/        init, status, branch, diff (working tree, staged, range)
│   ├── github/      gh wrapper: auth, repo create, issues, PRs, PR diff
│   ├── agent/       Agent interface + adapters: claude, codex, gemini
│   ├── intent/      wizard questions, intent.md read/write, bootstrap prompt
│   └── term/        pty session: spawn, resize, read/write, close
├── docs/superpowers/specs/
├── .github/workflows/ci.yml      go vet, go test, golangci-lint
├── .gitignore  README.md  LICENSE
```

`internal/` is used so nothing is importable from outside this module. That is
a Go language rule, not a convention, and it keeps the public surface zero
while learning.

---

## 4. Core packages

### 4.1 `internal/exec` — command runner

Everything that shells out goes through one small interface so tests can
substitute canned output.

```go
type Runner interface {
    Run(ctx context.Context, dir string, name string, args ...string) (stdout, stderr []byte, err error)
}
```

- `Real` wraps `os/exec`.
- `Fake` records calls and returns scripted results keyed by command.

### 4.2 `internal/config`

- Resolves `$XDG_CONFIG_HOME/espalier` (default `~/.config/espalier`).
- `settings.json`: default agent, usage display preferences (which windows to
  show), UI preferences.
- Load with defaults on missing file; save atomically (write temp, rename).

### 4.3 `internal/project`

```go
type Project struct {
    ID        string    // random short id
    Name      string
    Path      string    // absolute
    Remote    *Remote   // nil if no GitHub remote
    Agent     string    // agent ID override; "" = settings default
    AddedAt   time.Time
}
type Remote struct{ Owner, Repo string }
```

- Registry is `projects.json` in the config dir. Add, remove, list, get.
- `Open(path)`: validates the directory, detects git (`git rev-parse`), parses
  `origin` URL into `Remote` (supports https and ssh GitHub URLs), registers.
- `Create(opts)`: makes the directory, `git init -b main`, optional README,
  optional `gh repo create --source . --private|--public --push`, registers.
- Reads/writes `.espalier/project.toml` in the repo: chosen agent,
  `instructions_bootstrapped_at`.

### 4.4 `internal/gitx`

Thin, typed wrappers over `git`: `IsRepo`, `CurrentBranch`, `Status`,
`Branches`, `Diff(opts)` where `opts` selects working tree / staged / a
`base...head` range. Returns unified diff text; parsing into files and hunks
lives here too (`ParseUnified`), because both the desktop review view and
"send hunk to agent" need it.

### 4.5 `internal/github`

Wraps `gh` with `--json` output and decodes into Go structs.

- `AuthStatus()` → logged-in user or a typed `ErrNotAuthenticated`.
- `CreateRepo(dir, name, visibility)`.
- `ListIssues(remote, state)`, `GetIssue(remote, n)`, `CreateIssue(remote, title, body, labels)`.
- `ListPRs(remote, state)`, `GetPR(remote, n)`, `PRDiff(remote, n)`.

Errors from `gh` are wrapped with the command that ran so the UI can show
something useful. If `gh` is missing: `ErrGHNotInstalled` with an install hint.

### 4.6 `internal/agent`

```go
type Agent interface {
    ID() string                  // "claude" | "codex" | "gemini"
    DisplayName() string
    Detect() (path string, ok bool)       // exec.LookPath
    InstructionFile() string              // "CLAUDE.md" | "AGENTS.md" | "GEMINI.md"
    Command(ctx context.Context, p Project, o LaunchOptions) *exec.Cmd
    Usage(ctx context.Context) (*Usage, error)   // may return ErrUsageUnsupported
}

type LaunchOptions struct {
    OpeningPrompt string   // e.g. "Work on issue #12: …" or the bootstrap prompt
    Resume        bool
}

type Usage struct {
    Windows []UsageWindow   // e.g. five-hour, seven-day, weekly-scoped
    FetchedAt time.Time
}
type UsageWindow struct {
    Kind     string   // stable key used in settings to show/hide
    Label    string
    Percent  float64
    ResetsAt time.Time
}
```

`Registry` lists all adapters and which are installed.

**v1 adapters**

- **claude** — `claude` binary. Opening prompt is passed as the positional
  argument. Usage: GET `https://api.anthropic.com/api/oauth/usage` with the
  bearer token from `~/.claude/.credentials.json`, cached for 4 minutes, same
  as the author's statusline. Undocumented endpoint: on any failure, return
  `ErrUsageUnavailable`; the UI hides the panel and shows a tooltip.
- **codex** — `codex` binary. Instruction file `AGENTS.md`. Usage: unsupported
  in v1.
- **gemini** — `gemini` binary. Instruction file `GEMINI.md`. Usage:
  unsupported in v1.

Adding an agent = one file implementing the interface + a registry entry.

### 4.7 `internal/intent`

The per-project "what do I want from this project" layer.

- **Wizard questions** (v1 set, each with fixed choices plus free text):
  1. Workflow: issue-driven / free-form / teaching-first.
  2. Autonomy: ask before every change / ask before commits & pushes / may
     commit, never push / may push to branches, never main.
  3. Testing: required for every change / when touching logic / none.
  4. Commit style: conventional commits / plain / project convention (text).
  5. Branching: `feature/<issue>-<slug>` / `<user>/<slug>` / custom.
  6. Teaching mode: explain and let me write / write but explain / just do it.
  7. Anything else (free text).
- Writes `.espalier/intent.md`: YAML front matter with the structured
  answers, then the free text. Human-editable.
- **Bootstrap**: builds the opening prompt that tells the agent to read
  `intent.md` and write its instruction file (`CLAUDE.md` etc.) for this
  project, then stop. The UI must show a consent screen first:
  *"This will run <agent> once to write the instruction doc from your answers.
  It uses your <agent> limits."* On completion the app records
  `instructions_bootstrapped_at`. Re-running is allowed any time.

### 4.8 `internal/term`

Desktop-only pty layer (the TUI launches the agent in the foreground and
resumes itself when it exits).

- `Start(cmd *exec.Cmd, cols, rows) (*Session, error)` using `creack/pty`.
- `Session.Read` stream → frontend; `Session.Write(bytes)` ← frontend;
  `Resize(cols, rows)`; `Close()`.
- One session per project tab. Closing a tab kills the process group.

---

## 5. Frontends

### 5.1 TUI (`cmd/espalier`)

Screens, each a Bubble Tea model composed by a root model:

1. **Projects** — list from registry; `a` add existing, `n` new, `enter` open.
2. **Project** — header (name, branch, remote, agent), issue list with filter
   open/closed, `c` create issue, `l` launch agent, `i` launch on selected
   issue, `u` toggle usage line.
3. **Usage** — bars for each enabled window, same look as the author's
   statusline.
4. **Launch** — the TUI hands the terminal to the agent process (`tea.Exec`),
   then resumes when it exits.

No wizard, no diff view. If a project has no `intent.md`, the TUI shows a
hint to run the wizard in the desktop app.

### 5.2 Desktop (`cmd/espalier-desktop`)

Wails v3. Go side exposes services; frontend is plain TypeScript + a small
framework (author's choice; Svelte or vanilla recommended to keep the learning
load on Go, not on a frontend stack).

Views:

1. **Projects** sidebar — same registry.
2. **Project home** — issues, PRs, usage, agent picker, "Set up instructions"
   (wizard) and "Launch".
3. **Setup wizard** — the questions from §4.7 → writes `intent.md` → consent
   screen → bootstrap in the terminal pane.
4. **Terminal pane** — xterm.js bound to a `term.Session`. Multiple tabs.
5. **Review** —
   - Source selector: working tree, staged, branch vs base, or a PR.
   - File list + unified/side-by-side diff rendering.
   - Click a line to add a comment. Comments are kept in memory for the
     session (v1), optionally saved to `.espalier/review-notes.md`.
   - **Send to agent**: composes a prompt from selected hunks + comments and
     writes it to the active terminal session's stdin. If no session is
     running, it launches one with that as the opening prompt.
6. **Settings** — default agent, which usage windows to show, theme.

Packaging: `wails3 package` → Linux binary + `.desktop` file + icon. The
`.desktop` entry is what lets the app be pinned to the taskbar.

---

## 6. Key flows

**Add existing project**
pick folder → `gitx.IsRepo` → parse remote → register → open Project home.
If the remote is GitHub and `gh` is authenticated, issues load; otherwise the
issues panel shows why (not a GitHub remote / `gh` not logged in).

**Create project**
name + path + visibility → mkdir, `git init -b main`, README → if GitHub:
`gh repo create` with `--source` and `--push` → register.

**First launch on a project (desktop)**
no `intent.md` → prompt to run wizard → wizard → consent → bootstrap session
in the terminal pane → agent writes `CLAUDE.md` → user reviews/edits file →
normal launches from here on.

**Work on an issue**
select issue → "Launch on issue" → opening prompt =
`Work on GitHub issue #N: <title>\n\n<body>\n\nFollow the project instructions.`
→ agent runs in terminal pane / foreground.

**Review and iterate (desktop)**
Review view → pick working tree → comment on lines → Send to agent → agent
makes changes → diff refreshes (file watcher or manual refresh in v1) → repeat
→ when happy, user commits/pushes via the agent or their own shell.

---

## 7. Error handling

- Missing tool (`git`, `gh`, agent binary): typed sentinel errors; UIs show an
  install hint, never a stack trace.
- `gh` not authenticated: show `gh auth login` instruction; GitHub panels
  disabled, everything else works.
- Usage endpoint failure or schema change: hide the panel; log at debug.
- Pty process dies: tab shows exit code and a relaunch button.
- Registry or settings file corrupt: back it up as `.bak`, start fresh, tell
  the user.
- All core functions take a `context.Context` so the UI can cancel.

---

## 8. Testing

- **Unit tests with the fake runner**: every `gitx`, `github`, and `agent`
  call is tested against recorded `gh --json` / `git` output stored as fixture
  files under `testdata/`. No network, no real git in unit tests.
- **Golden tests** for `intent.md` rendering and the bootstrap prompt.
- **Table-driven tests** for remote URL parsing (https, ssh, with/without
  `.git`).
- **One integration test** (build tag `integration`) that creates a temp git
  repo and runs the real `git` binary through `gitx`.
- UI: Bubble Tea models tested by sending messages and asserting on `View()`
  output for the main screens. Desktop frontend gets manual testing in v1.
- CI: GitHub Actions runs `go vet`, `golangci-lint`, `go test ./...` on every
  push and PR.

---

## 9. Milestones

Each is a GitHub milestone; each bullet becomes an issue.

- **M0 Foundation** — repo, module, CI, `.gitignore`, README, this spec
  committed, `internal/exec` with fake runner.
- **M1 Core** — `config`, `project` registry + open/create, `gitx`, `github`
  (auth, issues, repo create), `agent` interface + claude adapter with
  detection and launch. Small throwaway `cmd/espalier-cli` for poking at it is
  allowed and deleted later.
- **M2 TUI** — projects, project home with issues, launch agent, usage line.
- **M3 Instructions** — wizard questions, `intent.md`, bootstrap prompt +
  consent. Exposed in the TUI as "run the wizard in desktop" hint only; the
  wizard UI lands in M4.
- **M4 Desktop shell** — Wails v3 app, projects + issues views, wizard,
  embedded terminal with pty, packaging with `.desktop` entry.
- **M5 Review** — diff sources, rendering, line comments, send-to-agent,
  PR list/view/diff.
- **M6 More agents + polish** — codex and gemini adapters, their usage
  adapters if a source exists, settings for usage windows, file watcher for
  diff refresh.

---

## 10. Open items (decide when reached, not now)

- Frontend framework for the Wails app (Svelte vs vanilla TS).
- Whether to migrate `github` from `gh` to the REST API via `google/go-github`
  once the author has learned the model through `gh`.
- Whether review comments should become real GitHub PR review comments.
- Session history: read each agent's own session store, or just rely on the
  agent's `--resume`.
