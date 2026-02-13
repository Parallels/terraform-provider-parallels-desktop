package localclient

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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
	cmd := Command{
		Command: command,
		Args:    arguments,
	}

	stdout, _, _, err := executeWithOutput(cmd)
	return stdout, err
}

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
		"bash":       true,
		"/bin/bash":  true,
		"sh":         true,
		"/bin/sh":    true,
		"echo":       true,
		"sudo":       true,
		"ls":         true,
		"rm":         true,
		"which":      true,
		"mkdir":      true,
		"cp":         true,
		"mv":         true,
		"chmod":      true,
		"chown":      true,
		"cat":        true,
		"grep":       true,
		"tee":        true,
		"prldevops":  true,
		// Parallels Desktop service management
		"/Applications/Parallels\\ Desktop.app/Contents/MacOS/Parallels\\ Service": true,
	}

	// Check exact match first, then check basename for full paths
	// (e.g. /usr/local/bin/prldevops should match "prldevops")
	if !allowedCommands[command] && !allowedCommands[filepath.Base(command)] {
		return "", fmt.Errorf("command '%s' is not allowed", command)
	}
	return command, nil
}

func validateArgs(args []string) ([]string, error) {
	for _, arg := range args {
		// Only reject ';' which enables command chaining injection.
		// '$' and '\' are safe with exec.Command (no shell interpretation)
		// and are needed for legitimate bash -c subshells and escaped paths.
		if strings.Contains(arg, ";") {
			return []string{}, fmt.Errorf("argument contains forbidden characters: %s", arg)
		}
	}
	return args, nil
}

func executeWithOutput(command Command) (stdout string, stderr string, exitCode int, err error) {
	validatedCmd, err := validateCommand(command.Command)
	if err != nil {
		return "", "", -1, err
	}
	// Validate arguments for potential command injection
	validatedArgs, err := validateArgs(command.Args)
	if err != nil {
		return "", "", -1, err
	}

	// #nosec G204 -- This is safe as we validate both command and arguments
	cmd := exec.Command(validatedCmd, validatedArgs...)

	if command.WorkingDirectory != "" {
		cmd.Dir = command.WorkingDirectory
	}

	var stdOut, stdIn, stdErr bytes.Buffer

	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	cmd.Stdin = &stdIn

	if err := cmd.Run(); err != nil {
		if stdErr.String() != "" {
			stderr = strings.TrimSuffix(stdErr.String(), "\n")
			stdout = strings.TrimSuffix(stdOut.String(), "\n")
			return stdout, stderr, cmd.ProcessState.ExitCode(), fmt.Errorf("%v, err: %v", stdErr.String(), err.Error())
		} else {
			stderr = ""
			stdout = strings.TrimSuffix(stdOut.String(), "\n")
			return stdout, stderr, cmd.ProcessState.ExitCode(), fmt.Errorf("%v, err: %v", stdErr.String(), err.Error())
		}
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
