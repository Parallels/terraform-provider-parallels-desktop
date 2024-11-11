package models

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMachineStateResourceModel describes the resource data model.
type VirtualMachineStateResourceModelV1 struct {
	Authenticator *authenticator.Authentication `tfsdk:"authenticator"`
	Orchestrator  types.String                  `tfsdk:"orchestrator"`
	Host          types.String                  `tfsdk:"host"`
	ID            types.String                  `tfsdk:"id"`
	Operation     types.String                  `tfsdk:"operation"`
	CurrentState  types.String                  `tfsdk:"current_state"`
}
