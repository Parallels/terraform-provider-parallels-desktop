package models

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"
	"terraform-provider-parallels-desktop/internal/schemas/vmconfig"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// CloneVmResourceModelV0 describes the resource data model.
type CloneVmResourceModelV1 struct {
	Authenticator        *authenticator.Authentication              `tfsdk:"authenticator"`
	Host                 types.String                               `tfsdk:"host"`
	Orchestrator         types.String                               `tfsdk:"orchestrator"`
	ID                   types.String                               `tfsdk:"id"`
	OsType               types.String                               `tfsdk:"os_type"`
	BaseVmId             types.String                               `tfsdk:"base_vm_id"`
	ExternalIp           types.String                               `tfsdk:"external_ip"`
	InternalIp           types.String                               `tfsdk:"internal_ip"`
	Name                 types.String                               `tfsdk:"name"`
	Owner                types.String                               `tfsdk:"owner"`
	Path                 types.String                               `tfsdk:"path"`
	Specs                *vmspecs.VmSpecs                           `tfsdk:"specs"`
	PostProcessorScripts []*postprocessorscript.PostProcessorScript `tfsdk:"post_processor_script"`
	OnDestroyScript      []*postprocessorscript.PostProcessorScript `tfsdk:"on_destroy_script"`
	SharedFolder         []*sharedfolder.SharedFolder               `tfsdk:"shared_folder"`
	Config               *vmconfig.VmConfig                         `tfsdk:"config"`
	PrlCtl               []*prlctl.PrlCtlCmd                        `tfsdk:"prlctl"`
	RunAfterCreate       types.Bool                                 `tfsdk:"run_after_create"`
	Timeouts             timeouts.Value                             `tfsdk:"timeouts"`
	ForceChanges         types.Bool                                 `tfsdk:"force_changes"`
	KeepRunning          types.Bool                                 `tfsdk:"keep_running"`
	KeepAfterError       types.Bool                                 `tfsdk:"keep_after_error"`
	ReverseProxyHosts    []*reverseproxy.ReverseProxyHost           `tfsdk:"reverse_proxy_host"`
}
