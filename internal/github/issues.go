package github

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Sean-Ouellette/espalier/internal/project"
)

// State filters issue and PR listings.
type State string

// Values accepted by gh's --state flag.
const (
	StateOpen   State = "open"
	StateClosed State = "closed"
	StateAll    State = "all"
)

// issueFields is the --json field list; it is also part of the exec.Fake
// key, so tests reference the same constant.
const issueFields = "number,title,body,state,labels,url,updatedAt"

// Issue is a GitHub issue as reported by gh.
type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"` // "OPEN" or "CLOSED", as gh reports it
	Labels    []string  `json:"labels"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UnmarshalJSON flattens gh's label objects ([{"name": ...}]) into
// Issue.Labels. It decodes into an alias of Issue so the standard decoder
// handles every other field; without the alias, calling json.Unmarshal on
// Issue would recurse back into this method.
func (i *Issue) UnmarshalJSON(data []byte) error {
	type plain Issue
	var wire struct {
		plain
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*i = Issue(wire.plain)
	i.Labels = make([]string, len(wire.Labels))
	for n, l := range wire.Labels {
		i.Labels[n] = l.Name
	}
	return nil
}

func repoFlag(r project.Remote) string {
	return r.Owner + "/" + r.Repo
}

// ListIssues returns up to 100 issues in the given state, newest first.
func (g *GH) ListIssues(ctx context.Context, remote project.Remote, state State) ([]Issue, error) {
	out, err := g.run(ctx, "issue", "list", "--repo", repoFlag(remote),
		"--state", string(state), "--limit", "100", "--json", issueFields)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("github: decode issue list: %w", err)
	}
	return issues, nil
}

// GetIssue returns one issue by number.
func (g *GH) GetIssue(ctx context.Context, remote project.Remote, number int) (Issue, error) {
	out, err := g.run(ctx, "issue", "view", strconv.Itoa(number),
		"--repo", repoFlag(remote), "--json", issueFields)
	if err != nil {
		return Issue{}, err
	}
	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return Issue{}, fmt.Errorf("github: decode issue %d: %w", number, err)
	}
	return issue, nil
}

// issueURL matches the URL gh prints after creating an issue.
var issueURL = regexp.MustCompile(`/issues/(\d+)\s*$`)

// CreateIssue opens an issue and returns it as GitHub now reports it.
// gh issue create prints the new issue's URL rather than JSON, so the number
// is parsed from that URL and the issue fetched with GetIssue.
func (g *GH) CreateIssue(ctx context.Context, remote project.Remote, title, body string, labelNames []string) (Issue, error) {
	args := []string{"issue", "create", "--repo", repoFlag(remote), "--title", title, "--body", body}
	for _, l := range labelNames {
		args = append(args, "--label", l)
	}
	out, err := g.run(ctx, args...)
	if err != nil {
		return Issue{}, err
	}

	m := issueURL.FindStringSubmatch(strings.TrimSpace(string(out)))
	if m == nil {
		return Issue{}, fmt.Errorf("github: gh issue create: no issue URL in output %q", strings.TrimSpace(string(out)))
	}
	number, err := strconv.Atoi(m[1])
	if err != nil {
		return Issue{}, fmt.Errorf("github: parse issue number from %q: %w", m[1], err)
	}
	return g.GetIssue(ctx, remote, number)
}
