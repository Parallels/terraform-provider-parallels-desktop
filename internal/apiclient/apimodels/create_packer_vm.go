package apimodels

type CreatePackerVmRequest struct {
	Template     string `json:"template"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	Memory       string `json:"memory"`
	Cpu          string `json:"cpu"`
	Disk         string `json:"disk"`
	DesiredState string `json:"desiredState"`
}
