package apimodels

import "github.com/goccy/go-json"

type VmConfigRequest struct {
	Owner      string                      `json:"owner"`
	Operations []*VmConfigRequestOperation `json:"operations"`
}

func NewVmConfigRequest(owner string) *VmConfigRequest {
	return &VmConfigRequest{
		Owner:      owner,
		Operations: make([]*VmConfigRequestOperation, 0),
	}
}

func (r *VmConfigRequest) AddOperation(operation *VmConfigRequestOperation) {
	r.Operations = append(r.Operations, operation)
}

func (r *VmConfigRequest) HasErrors() bool {
	for _, operation := range r.Operations {
		if operation.Error != nil {
			return true
		}
	}

	return false
}

func (r *VmConfigRequest) HasChanges() bool {
	return len(r.Operations) > 0
}

func (r *VmConfigRequest) String() string {
	std, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return ""
	}

	return string(std)
}
