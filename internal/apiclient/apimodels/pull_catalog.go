package apimodels

type PullCatalogRequest struct {
	ID          string `json:"id,omitempty"`
	MachineName string `json:"machine_name,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Connection  string `json:"connection,omitempty"`
	Path        string `json:"path,omitempty"`
}

type PullCatalogResponse struct {
	ID string `json:"id,omitempty"`
}
