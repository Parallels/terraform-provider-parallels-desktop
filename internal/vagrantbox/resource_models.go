package vagrantbox

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"
	"terraform-provider-parallels-desktop/internal/schemas/vmconfig"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMachineStateResourceModel describes the resource data model.
type VagrantBoxResourceModel struct {
	Authenticator         *authenticator.Authentication              `tfsdk:"authenticator"`
	Host                  types.String                               `tfsdk:"host"`
	ID                    types.String                               `tfsdk:"id"`
	OsType                types.String                               `tfsdk:"os_type"`
	BoxName               types.String                               `tfsdk:"box_name"`
	BoxVersion            types.String                               `tfsdk:"box_version"`
	VagrantFilePath       types.String                               `tfsdk:"vagrant_file_path"`
	CustomVagrantConfig   types.String                               `tfsdk:"custom_vagrant_config"`
	CustomParallelsConfig types.String                               `tfsdk:"custom_parallels_config"`
	Name                  types.String                               `tfsdk:"name"`
	Owner                 types.String                               `tfsdk:"owner"`
	RunAfterCreate        types.Bool                                 `tfsdk:"run_after_create"`
	Timeouts              timeouts.Value                             `tfsdk:"timeouts"`
	Specs                 *vmspecs.VmSpecs                           `tfsdk:"specs"`
	PostProcessorScripts  []*postprocessorscript.PostProcessorScript `tfsdk:"post_processor_script"`
	OnDestroyScript       []*postprocessorscript.PostProcessorScript `tfsdk:"on_destroy_script"`
	SharedFolder          []*sharedfolder.SharedFolder               `tfsdk:"shared_folder"`
	ForceChanges          types.Bool                                 `tfsdk:"force_changes"`
	Config                *vmconfig.VmConfig                         `tfsdk:"config"`
	PrlCtl                []*prlctl.PrlCtlCmd                        `tfsdk:"prlctl"`
}
