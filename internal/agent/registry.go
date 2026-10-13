package agent

// Registry lists the known adapters in display order.
type Registry struct {
	agents []Agent
}

// NewRegistry returns the v1 adapters: claude, codex, gemini.
func NewRegistry() *Registry {
	return &Registry{agents: []Agent{Claude{}, Codex{}, Gemini{}}}
}

// All returns every adapter, installed or not.
func (r *Registry) All() []Agent {
	out := make([]Agent, len(r.agents))
	copy(out, r.agents)
	return out
}

// Installed returns the adapters whose binary is on PATH.
func (r *Registry) Installed() []Agent {
	var out []Agent
	for _, a := range r.agents {
		if _, ok := a.Detect(); ok {
			out = append(out, a)
		}
	}
	return out
}

// Get returns the adapter with id.
func (r *Registry) Get(id string) (Agent, bool) {
	for _, a := range r.agents {
		if a.ID() == id {
			return a, true
		}
	}
	return nil, false
}
