package clientmodels

type NewPackerTemplateVmRequest struct {
	Template string            `json:"template"`
	Name     string            `json:"name"`
	Owner    string            `json:"owner"`
	Specs    map[string]string `json:"specs"`
}

type NewPackerTemplateVmResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Owner        string `json:"owner"`
	CurrentState string `json:"current_state"`
}
