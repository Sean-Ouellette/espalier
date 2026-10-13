package exec

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"git", nil, "git"},
		{"git", []string{"status"}, "git status"},
		{"git", []string{"status", "--porcelain"}, "git status --porcelain"},
	}
	for _, tt := range tests {
		if got := Key(tt.name, tt.args...); got != tt.want {
			t.Errorf("Key(%q, %q) = %q, want %q", tt.name, tt.args, got, tt.want)
		}
	}
}

func TestFakeRunScripted(t *testing.T) {
	wantErr := errors.New("boom")
	f := &Fake{Results: map[string]Result{
		Key("git", "status"): {Stdout: []byte("out"), Stderr: []byte("err"), Err: wantErr},
	}}

	stdout, stderr, err := f.Run(context.Background(), "/repo", "git", "status")

	if string(stdout) != "out" {
		t.Errorf("stdout = %q, want %q", stdout, "out")
	}
	if string(stderr) != "err" {
		t.Errorf("stderr = %q, want %q", stderr, "err")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestFakeRunUnscripted(t *testing.T) {
	f := &Fake{}

	_, _, err := f.Run(context.Background(), "", "git", "status")

	if err == nil {
		t.Fatal("expected error for unscripted command, got nil")
	}
	if !strings.Contains(err.Error(), "git status") {
		t.Errorf("error %q should name the command", err)
	}
}

func TestFakeRecordsCalls(t *testing.T) {
	f := &Fake{}
	ctx := context.Background()

	_, _, _ = f.Run(ctx, "/a", "git", "status")
	_, _, _ = f.Run(ctx, "/b", "gh", "issue", "list")

	if len(f.Calls) != 2 {
		t.Fatalf("len(Calls) = %d, want 2", len(f.Calls))
	}
	want := []Call{
		{Dir: "/a", Name: "git", Args: []string{"status"}},
		{Dir: "/b", Name: "gh", Args: []string{"issue", "list"}},
	}
	for i, w := range want {
		got := f.Calls[i]
		if got.Dir != w.Dir || got.Name != w.Name || !slices.Equal(got.Args, w.Args) {
			t.Errorf("Calls[%d] = %+v, want %+v", i, got, w)
		}
	}
}
