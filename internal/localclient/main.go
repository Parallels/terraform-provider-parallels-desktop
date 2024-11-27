package localclient

import (
	"bytes"
	"fmt"
	"os/exec"
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

func executeWithOutput(command Command) (stdout string, stderr string, exitCode int, err error) {
	cmd := exec.Command(command.Command, command.Args...)
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
