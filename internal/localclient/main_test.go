package localclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLiteralArgumentsAndInput(t *testing.T) {
	c := NewLocalClient()
	value := "a'\"$HOME `echo bad`; \\ spaces"
	output, err := c.RunCommandContext(context.Background(), "echo", []string{value})
	if err != nil || output != value {
		t.Fatalf("%q %v", output, err)
	}
	output, err = c.RunCommandInput(context.Background(), "cat", nil, strings.NewReader(value))
	if err != nil || output != value {
		t.Fatalf("%q %v", output, err)
	}
}

func TestCancellationAndStartFailure(t *testing.T) {
	c := NewLocalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.RunCommandContext(ctx, "/bin/sh", []string{"-c", "exec sleep 10"})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := c.RunCommandContext(context.Background(), "/does-not-exist/git", nil); err == nil {
		t.Fatal("missing executable succeeded")
	}
}

func TestErrorDoesNotIncludeOutput(t *testing.T) {
	secret := "private-password"
	_, err := NewLocalClient().RunCommandContext(context.Background(), "/bin/sh", []string{"-c", "echo \"$1\" >&2; exit 1", "test", secret})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("secret leaked: %v", err)
	}
}
