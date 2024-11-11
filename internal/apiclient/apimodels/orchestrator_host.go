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
	ID                       string                    `json:"id"`
	Enabled                  bool                      `json:"enabled"`
	Host                     string                    `json:"host"`
	Architecture             string                    `json:"architecture"`
	CpuModel                 string                    `json:"cpu_model"`
	OsVersion                string                    `json:"os_version,omitempty"`
	OsName                   string                    `json:"os_name,omitempty"`
	ExternalIpAddress        string                    `json:"external_ip_address,omitempty"`
	DevOpsVersion            string                    `json:"devops_version,omitempty"`
	Description              string                    `json:"description,omitempty"`
	Tags                     []string                  `json:"tags,omitempty"`
	State                    string                    `json:"state,omitempty"`
	ParallelsDesktopVersion  string                    `json:"parallels_desktop_version,omitempty"`
	ParallelsDesktopLicensed bool                      `json:"parallels_desktop_licensed,omitempty"`
	IsReverseProxyEnabled    bool                      `json:"is_reverse_proxy_enabled"`
	ReverseProxy             *HostReverseProxy         `json:"reverse_proxy,omitempty"`
	Resources                OrchestratorHostResources `json:"resources"`
	RequiredClaims           []string                  `json:"required_claims,omitempty"`
	RequiredRoles            []string                  `json:"required_roles,omitempty"`
}

type OrchestratorHostResources struct {
	TotalAppleVms    int64   `json:"total_apple_vms,omitempty"`
	PhysicalCpuCount int64   `json:"physical_cpu_count,omitempty"`
	LogicalCpuCount  int64   `json:"logical_cpu_count"`
	MemorySize       float64 `json:"memory_size,omitempty"`
	DiskSize         float64 `json:"disk_size,omitempty"`
	FreeDiskSize     float64 `json:"free_disk_size,omitempty"`
}

type HostReverseProxy struct {
	Enabled bool               `json:"enabled,omitempty"`
	Host    string             `json:"host,omitempty"`
	Port    string             `json:"port,omitempty"`
	Hosts   []ReverseProxyHost `json:"hosts,omitempty"`
}
