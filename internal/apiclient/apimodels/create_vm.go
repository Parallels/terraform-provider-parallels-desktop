package apimodels

type CreateVmRequest struct {
	Name           string                  `json:"name"`
	Owner          string                  `json:"owner,omitempty"`
	PackerTemplate *CreatePackerVmRequest  `json:"packer_template,omitempty"`
	VagrantBox     *CreateVagrantVmRequest `json:"vagrant_box,omitempty"`
}

type CreateVmResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	CurrentState string `json:"current_state"`
}
