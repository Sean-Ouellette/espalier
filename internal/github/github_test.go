package github

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	xexec "github.com/Sean-Ouellette/espalier/internal/exec"
	"github.com/Sean-Ouellette/espalier/internal/project"
)

var espalier = project.Remote{Owner: "Sean-Ouellette", Repo: "espalier"}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func exitErr(t *testing.T) error {
	t.Helper()
	err := exec.Command("sh", "-c", "exit 1").Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	return ee
}

func TestAuthStatusLoggedIn(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "api", "user", "--jq", ".login"): {Stdout: []byte("Sean-Ouellette\n")},
	}}

	login, err := New(fake).AuthStatus(context.Background())
	if err != nil {
		t.Fatalf("AuthStatus() error: %v", err)
	}
	if login != "Sean-Ouellette" {
		t.Errorf("AuthStatus() = %q, want Sean-Ouellette", login)
	}
}

func TestAuthStatusNotLoggedIn(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "api", "user", "--jq", ".login"): {
			Stderr: []byte("To get started with GitHub CLI, please run:  gh auth login\n"),
			Err:    exitErr(t),
		},
	}}

	_, err := New(fake).AuthStatus(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("AuthStatus() error = %v, want ErrNotAuthenticated", err)
	}
}

func TestGHMissingReturnsSentinel(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "api", "user", "--jq", ".login"): {Err: exec.ErrNotFound},
	}}

	_, err := New(fake).AuthStatus(context.Background())
	if !errors.Is(err, ErrGHNotInstalled) {
		t.Errorf("AuthStatus() error = %v, want ErrGHNotInstalled", err)
	}
}

func TestListIssuesDecodesFixture(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "list", "--repo", "Sean-Ouellette/espalier", "--state", "all",
			"--limit", "100", "--json", issueFields): {Stdout: fixture(t, "issue_list.json")},
	}}

	issues, err := New(fake).ListIssues(context.Background(), espalier, StateAll)
	if err != nil {
		t.Fatalf("ListIssues() error: %v", err)
	}
	if len(issues) != 4 {
		t.Fatalf("got %d issues, want 4", len(issues))
	}
	first := issues[0]
	if first.Number != 4 || first.State != "OPEN" || first.Title != "internal/exec: Fake runner for tests" {
		t.Errorf("first issue = %+v", first)
	}
	if first.URL != "https://github.com/Sean-Ouellette/espalier/issues/4" {
		t.Errorf("URL = %q", first.URL)
	}
	if want := time.Date(2026, 10, 6, 20, 1, 9, 0, time.UTC); !first.UpdatedAt.Equal(want) {
		t.Errorf("UpdatedAt = %v, want %v", first.UpdatedAt, want)
	}
	if fake.Calls[0].Dir != "" {
		t.Errorf("gh ran in dir %q, want none (repo passed by --repo)", fake.Calls[0].Dir)
	}
}

func TestListIssuesWrapsGHError(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "list", "--repo", "Sean-Ouellette/espalier", "--state", "open",
			"--limit", "100", "--json", issueFields): {
			Stderr: []byte("GraphQL: Could not resolve to a Repository\n"), Err: exitErr(t),
		},
	}}

	_, err := New(fake).ListIssues(context.Background(), espalier, StateOpen)
	if err == nil {
		t.Fatal("ListIssues() error = nil, want failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "Could not resolve") || !strings.Contains(msg, "gh issue list") {
		t.Errorf("error %q should include gh's stderr and the command", msg)
	}
}

func TestGetIssueFlattensLabels(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "view", "4", "--repo", "Sean-Ouellette/espalier",
			"--json", issueFields): {Stdout: fixture(t, "issue_view.json")},
	}}

	issue, err := New(fake).GetIssue(context.Background(), espalier, 4)
	if err != nil {
		t.Fatalf("GetIssue() error: %v", err)
	}
	if issue.Number != 4 {
		t.Errorf("Number = %d, want 4", issue.Number)
	}
	if want := []string{"enhancement"}; !reflect.DeepEqual(issue.Labels, want) {
		t.Errorf("Labels = %v, want %v", issue.Labels, want)
	}
	if !strings.HasPrefix(issue.Body, "Fake implementation") {
		t.Errorf("Body = %q", issue.Body)
	}
}

func TestCreateIssueParsesNumberAndFetches(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "create", "--repo", "Sean-Ouellette/espalier",
			"--title", "Add thing", "--body", "Because.", "--label", "enhancement", "--label", "good first issue"): {
			Stdout: []byte("\nCreating issue in Sean-Ouellette/espalier\n\nhttps://github.com/Sean-Ouellette/espalier/issues/4\n"),
		},
		xexec.Key("gh", "issue", "view", "4", "--repo", "Sean-Ouellette/espalier",
			"--json", issueFields): {Stdout: fixture(t, "issue_view.json")},
	}}

	issue, err := New(fake).CreateIssue(context.Background(), espalier, "Add thing", "Because.",
		[]string{"enhancement", "good first issue"})
	if err != nil {
		t.Fatalf("CreateIssue() error: %v", err)
	}
	if issue.Number != 4 || issue.Title != "internal/exec: Fake runner for tests" {
		t.Errorf("CreateIssue() = %+v, want issue 4 from fixture", issue)
	}
	if len(fake.Calls) != 2 {
		t.Errorf("gh called %d times, want 2 (create, view)", len(fake.Calls))
	}
}

func TestCreateIssueWithoutLabelsOmitsFlag(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "create", "--repo", "Sean-Ouellette/espalier",
			"--title", "T", "--body", "B"): {
			Stdout: []byte("https://github.com/Sean-Ouellette/espalier/issues/4\n"),
		},
		xexec.Key("gh", "issue", "view", "4", "--repo", "Sean-Ouellette/espalier",
			"--json", issueFields): {Stdout: fixture(t, "issue_view.json")},
	}}

	if _, err := New(fake).CreateIssue(context.Background(), espalier, "T", "B", nil); err != nil {
		t.Fatalf("CreateIssue() error: %v", err)
	}
}

func TestCreateIssueUnparseableOutput(t *testing.T) {
	fake := &xexec.Fake{Results: map[string]xexec.Result{
		xexec.Key("gh", "issue", "create", "--repo", "Sean-Ouellette/espalier",
			"--title", "T", "--body", "B"): {Stdout: []byte("something unexpected\n")},
	}}

	_, err := New(fake).CreateIssue(context.Background(), espalier, "T", "B", nil)
	if err == nil {
		t.Fatal("CreateIssue() error = nil, want parse failure")
	}
}
