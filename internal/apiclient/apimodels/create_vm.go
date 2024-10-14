package apimodels

type CreateVmRequest struct {
	Name            string                        `json:"name"`
	Owner           string                        `json:"owner,omitempty"`
	Architecture    string                        `json:"architecture,omitempty"`
	PackerTemplate  *CreatePackerVmRequest        `json:"packer_template,omitempty"`
	VagrantBox      *CreateVagrantVmRequest       `json:"vagrant_box,omitempty"`
	CatalogManifest *CreateCatalogManifestRequest `json:"catalog_manifest,omitempty"`
}

type CreateVmResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	CurrentState string `json:"current_state"`
}
