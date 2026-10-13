package project

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/Sean-Ouellette/espalier/internal/config"
)

const registryFile = "projects.json"

// ErrAlreadyAdded means a project with the same path is already registered.
var ErrAlreadyAdded = errors.New("project: already added")

// CorruptError is returned by LoadRegistry when projects.json is unreadable;
// the registry still comes back empty and usable. It is the same type as
// config.CorruptError (a type alias), so errors.As works with either name.
type CorruptError = config.CorruptError

// Registry is the in-memory set of projects backed by projects.json.
// Mutations happen in memory; call Save to persist.
type Registry struct {
	path     string
	projects []Project
}

// LoadRegistry reads projects.json from dir. A missing file gives an empty
// registry. A corrupt file is backed up as projects.json.bak and an empty
// registry is returned alongside a *CorruptError.
func LoadRegistry(dir string) (*Registry, error) {
	reg := &Registry{path: filepath.Join(dir, registryFile)}
	if err := config.ReadJSON(reg.path, &reg.projects); err != nil {
		reg.projects = nil
		return reg, err
	}
	return reg, nil
}

// Save writes the registry atomically to projects.json.
func (r *Registry) Save() error {
	if r.projects == nil {
		// Encode as [] rather than null so the file stays a JSON array.
		return config.WriteJSON(r.path, []Project{})
	}
	return config.WriteJSON(r.path, r.projects)
}

// Add registers p. Two projects may not share a path.
func (r *Registry) Add(p Project) error {
	for _, q := range r.projects {
		if q.Path == p.Path {
			return fmt.Errorf("%w: %s", ErrAlreadyAdded, p.Path)
		}
	}
	r.projects = append(r.projects, p)
	return nil
}

// Remove deletes the project with id and reports whether it existed.
func (r *Registry) Remove(id string) bool {
	for i, p := range r.projects {
		if p.ID == id {
			r.projects = append(r.projects[:i], r.projects[i+1:]...)
			return true
		}
	}
	return false
}

// Get returns the project with id.
func (r *Registry) Get(id string) (Project, bool) {
	for _, p := range r.projects {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

// List returns a copy of the projects sorted by name, then path.
func (r *Registry) List() []Project {
	out := make([]Project, len(r.projects))
	copy(out, r.projects)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Path < out[j].Path
	})
	return out
}
