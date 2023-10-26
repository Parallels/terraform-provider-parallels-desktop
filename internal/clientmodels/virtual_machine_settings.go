package clientmodels

type VirtualMachineSetRequest struct {
	Owner      string                        `json:"owner"`
	Operations []*VirtualMachineSetOperation `json:"operations"`
}

type VirtualMachineSetOperation struct {
	Owner     string                              `json:"owner"`
	Group     string                              `json:"group"`
	Operation string                              `json:"operation"`
	Value     string                              `json:"value"`
	Options   []*VirtualMachineSetOperationOption `json:"options"`
	Error     error                               `json:"error"`
}

type VirtualMachineSetOperationOption struct {
	Flag  string `json:"flag"`
	Value string `json:"value"`
}

type VirtualMachineSetResponse struct {
	Operations []VirtualMachineSetOperationResponse `json:"operations"`
}

type VirtualMachineSetOperationResponse struct {
	Group     string `json:"group"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}
