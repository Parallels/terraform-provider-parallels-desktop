package apiclient

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
)

type HostConfig struct {
	IsOrchestrator       bool                          `json:"is_orchestrator"`
	Host                 string                        `json:"host"`
	HostId               string                        `json:"host_id"`
	MachineId            string                        `json:"machine_id"`
	License              string                        `json:"license"`
	DisableTlsValidation bool                          `json:"disable_tls_validation"`
	Authorization        *authenticator.Authentication `json:"authorization"`
}
