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

// VirtualMachineStateResourceModel describes the resource data model.
type RemoteVmResourceModelV2 struct {
	Authenticator        *authenticator.Authentication              `tfsdk:"authenticator"`
	Host                 types.String                               `tfsdk:"host"`
	HostUrl              types.String                               `tfsdk:"host_url"`
	Orchestrator         types.String                               `tfsdk:"orchestrator"`
	ID                   types.String                               `tfsdk:"id"`
	ExternalIp           types.String                               `tfsdk:"external_ip"`
	InternalIp           types.String                               `tfsdk:"internal_ip"`
	OrchestratorHostId   types.String                               `tfsdk:"orchestrator_host_id"`
	OsType               types.String                               `tfsdk:"os_type"`
	CatalogId            types.String                               `tfsdk:"catalog_id"`
	Version              types.String                               `tfsdk:"version"`
	Architecture         types.String                               `tfsdk:"architecture"`
	Name                 types.String                               `tfsdk:"name"`
	Owner                types.String                               `tfsdk:"owner"`
	CatalogConnection    types.String                               `tfsdk:"catalog_connection"`
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
	ReverseProxyHosts    []*reverseproxy.ReverseProxyHost           `tfsdk:"reverse_proxy_host"`
	KeepAfterError       types.Bool                                 `tfsdk:"keep_after_error"`
}
