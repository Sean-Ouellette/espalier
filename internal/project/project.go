// Package project defines registered projects and how to open a directory
// as one. Registry persistence lives in registry.go.
package project

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Sean-Ouellette/espalier/internal/gitx"
)

// ErrNotRepo means the directory exists but is not inside a git work tree.
var ErrNotRepo = errors.New("project: not a git repository")

// Project is a registered working directory.
type Project struct {
	ID      string    `json:"id"`               // random short id
	Name    string    `json:"name"`             // defaults to the directory name
	Path    string    `json:"path"`             // absolute
	Remote  *Remote   `json:"remote,omitempty"` // nil if no GitHub remote
	Agent   string    `json:"agent,omitempty"`  // agent ID override; "" = settings default
	AddedAt time.Time `json:"added_at"`
}

// Remote identifies a GitHub repository.
type Remote struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

// Open inspects path and builds a Project from it without registering it.
// It requires an existing directory inside a git work tree. The origin
// remote is parsed into Remote when it points at GitHub; a missing or
// non-GitHub remote leaves Remote nil and is not an error.
func Open(ctx context.Context, git *gitx.Git, path string) (Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Project{}, fmt.Errorf("project: resolve %s: %w", path, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return Project{}, fmt.Errorf("project: open %s: %w", abs, err)
	}
	if !info.IsDir() {
		return Project{}, fmt.Errorf("project: open %s: not a directory", abs)
	}

	isRepo, err := git.IsRepo(ctx, abs)
	if err != nil {
		return Project{}, err
	}
	if !isRepo {
		return Project{}, fmt.Errorf("%w: %s", ErrNotRepo, abs)
	}

	remote, err := detectRemote(ctx, git, abs)
	if err != nil {
		return Project{}, err
	}

	id, err := newID()
	if err != nil {
		return Project{}, err
	}
	return Project{
		ID:      id,
		Name:    filepath.Base(abs),
		Path:    abs,
		Remote:  remote,
		AddedAt: time.Now(),
	}, nil
}

// detectRemote reads origin and parses it as a GitHub URL. Only a failure to
// run git is an error; "no origin" and "not GitHub" both yield nil.
func detectRemote(ctx context.Context, git *gitx.Git, dir string) (*Remote, error) {
	url, err := git.RemoteURL(ctx, dir, "origin")
	if errors.Is(err, gitx.ErrNoRemote) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	owner, repo, ok := gitx.ParseGitHubRemote(url)
	if !ok {
		return nil, nil
	}
	return &Remote{Owner: owner, Repo: repo}, nil
}

// newID returns 8 hex characters from crypto/rand.
func newID() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("project: generate id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
