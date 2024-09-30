package apiclient

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
)

type HostConfig struct {
	Host                 string                        `json:"host"`
	MachineId            string                        `json:"machine_id"`
	License              string                        `json:"license"`
	DisableTlsValidation bool                          `json:"disable_tls_validation"`
	Authorization        *authenticator.Authentication `json:"authorization"`
}
