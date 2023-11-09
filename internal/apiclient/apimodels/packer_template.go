package apimodels

type PackerTemplate struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	PackerFolder   string            `json:"packer_folder"`
	Variables      map[string]string `json:"variables,omitempty"`
	Addons         []string          `json:"addons,omitempty"`
	Specs          map[string]string `json:"specs,omitempty"`
	Defaults       map[string]string `json:"defaults,omitempty"`
	Internal       bool              `json:"internal,omitempty"`
	UpdatedAt      string            `json:"updated_at,omitempty"`
	CreatedAt      string            `json:"created_at,omitempty"`
	RequiredRoles  []string          `json:"required_roles,omitempty"`
	RequiredClaims []string          `json:"required_claims,omitempty"`
}
