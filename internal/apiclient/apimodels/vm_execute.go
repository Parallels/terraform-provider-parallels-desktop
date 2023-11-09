package apimodels

type VmExecuteCommandRequest struct {
	Command string `json:"command"`
}

type VmExecuteCommandResponse struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}
