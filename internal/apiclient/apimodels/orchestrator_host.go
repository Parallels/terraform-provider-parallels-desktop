package apimodels

type OrchestratorHostRequest struct {
	Host           string                      `json:"host"`
	Description    string                      `json:"description,omitempty"`
	Tags           []string                    `json:"tags,omitempty"`
	Authentication *OrchestratorAuthentication `json:"authentication,omitempty"`
	RequiredClaims []string                    `json:"required_claims,omitempty"`
	RequiredRoles  []string                    `json:"required_roles,omitempty"`
}

type OrchestratorAuthentication struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	ApiKey   string `json:"api_key,omitempty"`
}

type OrchestratorHostResponse struct {
	ID           string   `json:"id"`
	Host         string   `json:"host"`
	Architecture string   `json:"architecture"`
	CpuModel     string   `json:"cpu_model"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	State        string   `json:"state,omitempty"`
}

type OrchestratorHost struct {
	ID           string                    `json:"id"`
	Enabled      bool                      `json:"enabled"`
	Host         string                    `json:"host"`
	Architecture string                    `json:"architecture"`
	CPUModel     string                    `json:"cpu_model"`
	Description  string                    `json:"description"`
	Tags         []string                  `json:"tags"`
	State        string                    `json:"state"`
	Resources    OrchestratorHostResources `json:"resources"`
}

type OrchestratorHostResources struct {
	LogicalCPUCount int64 `json:"logical_cpu_count"`
	MemorySize      int64 `json:"memory_size"`
	DiskSize        int64 `json:"disk_size"`
}
