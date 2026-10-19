package common

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DefaultVmReadyTimeout is how long we wait for a VM to report an internal IP,
// long enough to survive a guest first-boot reboot.
const DefaultVmReadyTimeout = 15 * time.Minute

// ParseVmReadyTimeout reads the optional vm_ready_timeout attribute (a Go
// duration such as "10m"), falling back to DefaultVmReadyTimeout when unset.
func ParseVmReadyTimeout(value types.String) (time.Duration, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return DefaultVmReadyTimeout, diagnostics
	}

	timeout, err := time.ParseDuration(value.ValueString())
	if err != nil {
		diagnostics.AddError("Invalid vm_ready_timeout", fmt.Sprintf("%q is not a valid duration: %v", value.ValueString(), err))
		return 0, diagnostics
	}
	if timeout <= 0 {
		diagnostics.AddError("Invalid vm_ready_timeout", "vm_ready_timeout must be greater than zero")
		return 0, diagnostics
	}

	return timeout, diagnostics
}
