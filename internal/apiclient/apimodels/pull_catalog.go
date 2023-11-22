package apimodels

type PullCatalogRequest struct {
	CatalogId      string `json:"catalog_id,omitempty"`
	Version        string `json:"version,omitempty"`
	MachineName    string `json:"machine_name,omitempty"`
	Owner          string `json:"owner,omitempty"`
	Connection     string `json:"connection,omitempty"`
	Path           string `json:"path,omitempty"`
	StartAfterPull bool   `json:"start_after_pull,omitempty"`
}

type PullCatalogResponse struct {
	ID string `json:"id,omitempty"`
}
