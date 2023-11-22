package apimodels

type VmConfigResponseOperation struct {
	Group     string `json:"group"`
	Operation string `json:"operation"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}
