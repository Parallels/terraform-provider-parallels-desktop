package remoteimage

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMachineStateResourceModel describes the resource data model.
type RemoteVmResourceModel struct {
	Authenticator        *authenticator.Authentication              `tfsdk:"authenticator"`
	Host                 types.String                               `tfsdk:"host"`
	ID                   types.String                               `tfsdk:"id"`
	CatalogId            types.String                               `tfsdk:"catalog_id"`
	Version              types.String                               `tfsdk:"version"`
	Name                 types.String                               `tfsdk:"name"`
	Owner                types.String                               `tfsdk:"owner"`
	Connection           types.String                               `tfsdk:"host_connection"`
	Path                 types.String                               `tfsdk:"path"`
	Specs                *vmspecs.VmSpecs                           `tfsdk:"specs"`
	PostProcessorScripts []*postprocessorscript.PostProcessorScript `tfsdk:"post_processor_script"`
	SharedFolder         []*sharedfolder.SharedFolder               `tfsdk:"shared_folder"`
	RunAfterCreate       types.Bool                                 `tfsdk:"run_after_create"`
	Timeouts             timeouts.Value                             `tfsdk:"timeouts"`
	ForceChanges         types.Bool                                 `tfsdk:"force_changes"`
}
