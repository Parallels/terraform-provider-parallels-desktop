package apimodels

import "encoding/json"

type VmConfigRequestOperationOption struct {
	Flag  string `json:"flag"`
	Value string `json:"value"`
}

type VmConfigRequestOperation struct {
	parent    *VmConfigRequest                  `json:"-"`
	Owner     string                            `json:"owner"`
	Group     string                            `json:"group"`
	Operation string                            `json:"operation"`
	Value     string                            `json:"value"`
	Options   []*VmConfigRequestOperationOption `json:"options"`
	Flags     []string                          `json:"flags"`
	Error     error                             `json:"error"`
}

func NewVmConfigRequestOperation(parent *VmConfigRequest) *VmConfigRequestOperation {
	return &VmConfigRequestOperation{
		parent:  parent,
		Options: make([]*VmConfigRequestOperationOption, 0),
		Flags:   make([]string, 0),
	}
}

func (r *VmConfigRequestOperation) WithOwner(owner string) *VmConfigRequestOperation {
	r.Owner = owner
	return r
}

func (r *VmConfigRequestOperation) WithGroup(group string) *VmConfigRequestOperation {
	r.Group = group
	return r
}

func (r *VmConfigRequestOperation) WithOperation(operation string) *VmConfigRequestOperation {
	r.Operation = operation
	return r
}

func (r *VmConfigRequestOperation) WithValue(value string) *VmConfigRequestOperation {
	r.Value = value
	return r
}

func (r *VmConfigRequestOperation) WithOption(flag, value string) *VmConfigRequestOperation {
	r.Options = append(r.Options, &VmConfigRequestOperationOption{
		Flag:  flag,
		Value: value,
	})
	return r
}

func (r *VmConfigRequestOperation) WithFlag(flag string) *VmConfigRequestOperation {
	r.Flags = append(r.Flags, flag)
	return r
}

func (r *VmConfigRequestOperation) WithError(err error) *VmConfigRequestOperation {
	r.Error = err
	return r
}

func (r *VmConfigRequestOperation) Append() *VmConfigRequest {
	r.parent.AddOperation(r)
	return r.parent
}

func (r *VmConfigRequestOperation) String() string {
	std, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return ""
	}

	return string(std)
}
