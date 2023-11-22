package apimodels

type SystemUsageResponse struct {
	CpuType        string          `json:"cpu_type,omitempty"`
	TotalAvailable SystemUsageItem `json:"total_available,omitempty"`
	TotalInUse     SystemUsageItem `json:"total_in_use,omitempty"`
	TotalReserved  SystemUsageItem `json:"total_reserved,omitempty"`
}

type SystemUsageItem struct {
	PhysicalCpuCount int64   `json:"physical_cpu_count,omitempty"`
	LogicalCpuCount  int64   `json:"logical_cpu_count,omitempty"`
	MemorySize       float64 `json:"memory_size,omitempty"`
	DiskSize         float64 `json:"disk_count,omitempty"`
}
