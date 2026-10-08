package exec

import (
	"context"
	"fmt"
	"strings"
)

// Result is the scripted outcome of one command.
type Result struct {
	Stdout, Stderr []byte
	Err            error
}

// Call records one invocation of Run.
type Call struct {
	Dir  string
	Name string
	Args []string
}

// Fake is a Runner that returns scripted results and records calls.
// Results is keyed by the command line as a single string,
// e.g. "git status --porcelain". See Key.
type Fake struct {
	Results map[string]Result
	Calls   []Call
}

// Compile-time check that *Fake satisfies Runner.
var _ Runner = (*Fake)(nil)

// Key builds the Results lookup key for a command.
func Key(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), " ")
}

// Run records the call and returns the scripted Result for it.
// An unscripted command returns an error naming the command.
func (f *Fake) Run(_ context.Context, dir, name string, args ...string) ([]byte, []byte, error) {
	f.Calls = append(f.Calls, Call{Dir: dir, Name: name, Args: args})

	res, ok := f.Results[Key(name, args...)]
	if !ok {
		return nil, nil, fmt.Errorf("exec.Fake: no scripted result for %q", Key(name, args...))
	}
	return res.Stdout, res.Stderr, res.Err
}
