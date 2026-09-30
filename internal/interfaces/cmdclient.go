package interfaces

import (
	"context"
	"io"
)

type CommandClient interface {
	RunCommand(command string, arguments []string) (string, error)
	Username() string
	Password() string
}

// ContextCommandClient executes literal arguments with cancellation support.
type ContextCommandClient interface {
	CommandClient
	Close() error
	RunCommandContext(context.Context, string, []string) (string, error)
	RunCommandInput(context.Context, string, []string, io.Reader) (string, error)
}
