package localclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Command struct {
	Command          string
	WorkingDirectory string
	Args             []string
}

type LocalClient struct{}

func NewLocalClient() *LocalClient {
	return &LocalClient{}
}

func (l *LocalClient) RunCommand(command string, arguments []string) (string, error) {
	return l.RunCommandContext(context.Background(), command, arguments)
}

func (l *LocalClient) RunCommandContext(ctx context.Context, command string, arguments []string) (string, error) {
	return l.RunCommandInput(ctx, command, arguments, nil)
}

func (l *LocalClient) RunCommandInput(ctx context.Context, command string, arguments []string, input io.Reader) (string, error) {
	stdout, _, _, err := executeWithOutput(ctx, Command{Command: command, Args: arguments}, input)
	return stdout, err
}
func (l *LocalClient) Close() error { return nil }

func validateCommand(command string) (string, error) {
	// Whitelist of allowed commands
	allowedCommands := map[string]bool{
		"prlctl":    true,
		"prlsrvctl": true,
		"brew":      true,
		"vagrant":   true,
		"packer":    true,
		"git":       true,
		"curl":      true,
		"wget":      true,
		"tar":       true,
		"unzip":     true,
		"zip":       true,
		"devops":    true,
		// Commands needed for local deployment (install_local = true)
		"bash":      true,
		"/bin/bash": true,
		"sh":        true,
		"/bin/sh":   true,
		"echo":      true,
		"sudo":      true,
		"ls":        true,
		"rm":        true,
		"which":     true,
		"mkdir":     true,
		"cp":        true,
		"mv":        true,
		"chmod":     true,
		"chown":     true,
		"cat":       true,
		"grep":      true,
		"tee":       true,
		"prldevops": true,
		// Parallels Desktop service management
		"/Applications/Parallels Desktop.app/Contents/MacOS/Parallels Service": true,
	}

	// Check exact match first, then check basename for full paths
	// (e.g. /usr/local/bin/prldevops should match "prldevops")
	if !allowedCommands[command] && !allowedCommands[filepath.Base(command)] {
		return "", fmt.Errorf("command '%s' is not allowed", command)
	}
	return command, nil
}

func executeWithOutput(ctx context.Context, command Command, input io.Reader) (stdout string, stderr string, exitCode int, err error) {
	validatedCmd, err := validateCommand(command.Command)
	if err != nil {
		return "", "", -1, err
	}
	// Arguments are literal argv. Explicit shell scripts are owned by callers.
	cmd := exec.CommandContext(ctx, validatedCmd, command.Args...) // #nosec G204 -- validatedCmd is allowlisted and arguments are passed as literal argv.
	cmd.WaitDelay = time.Second

	if command.WorkingDirectory != "" {
		cmd.Dir = command.WorkingDirectory
	}

	var stdOut, stdErr bytes.Buffer

	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	cmd.Stdin = input

	if err := cmd.Run(); err != nil {
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		// Do not include arbitrary remote/tool output in diagnostics: it can contain secrets.
		return stdOut.String(), stdErr.String(), code, fmt.Errorf("local command failed: %w", err)
	}

	stderr = ""
	stdout = strings.TrimSuffix(stdOut.String(), "\n")
	return stdout, stderr, cmd.ProcessState.ExitCode(), nil
}

func (l *LocalClient) Username() string {
	return ""
}

func (l *LocalClient) Password() string {
	return ""
}
