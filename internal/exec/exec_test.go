package exec

import (
	"context"
	"strings"
	"testing"
)

func TestRealRun(t *testing.T) {
	var r Real
	stdout, _, err := r.Run(context.Background(), "", "echo", "hello")

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	got := strings.TrimSpace(string(stdout))
	if got != "hello" {
		t.Errorf("expected hello, got %q", got)
	}
}

func TestRealRunFailure(t *testing.T) {
	var r Real
	_, _, err := r.Run(context.Background(), "", "false")

	if err == nil {
		t.Fatalf("expected error from false command, got nil")
	}
}
