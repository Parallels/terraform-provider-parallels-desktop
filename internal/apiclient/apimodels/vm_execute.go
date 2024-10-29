package apimodels

type PostScriptItem struct {
	Command              string            `json:"command"`
	VirtualMachineId     string            `json:"virtual_machine_id"`
	OS                   string            `json:"os"`
	EnvironmentVariables map[string]string `json:"environment_variables,omitempty"`
}

type VmExecuteCommandRequest struct {
	Command              string            `json:"command"`
	EnvironmentVariables map[string]string `json:"environment_variables,omitempty"`
}

type VmExecuteCommandResponse struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}
