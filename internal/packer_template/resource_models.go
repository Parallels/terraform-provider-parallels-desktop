package packertemplate

import (
	"terraform-provider-parallels/internal/models"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMachineStateResourceModel describes the resource data model.
type PackerVirtualMachineResourceModel struct {
	Authenticator  *models.AuthorizationModel   `tfsdk:"authenticator"`
	Host           types.String                 `tfsdk:"host"`
	ID             types.String                 `tfsdk:"id"`
	OsType         types.String                 `tfsdk:"os_type"`
	Template       types.String                 `tfsdk:"template"`
	Name           types.String                 `tfsdk:"name"`
	Owner          types.String                 `tfsdk:"owner"`
	Specs          *VirtualMachineResourceSpecs `tfsdk:"specs"`
	RunAfterCreate types.Bool                   `tfsdk:"run_after_create"`
	Timeouts       timeouts.Value               `tfsdk:"timeouts"`
}

type VirtualMachineResourceSpecs struct {
	CpuCount   types.String `tfsdk:"cpu_count"`
	MemorySize types.String `tfsdk:"memory_size"`
	DiskSize   types.String `tfsdk:"disk_size"`
}
