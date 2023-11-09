package apimodels

type CreateVagrantVmRequest struct {
	Box                   string `json:"box"`
	Version               string `json:"version"`
	Owner                 string `json:"owner"`
	Name                  string `json:"name"`
	CustomVagrantConfig   string `json:"custom_vagrant_config"`
	CustomParallelsConfig string `json:"custom_parallels_config"`
}
