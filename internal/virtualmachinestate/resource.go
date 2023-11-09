package virtualmachinestate

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VirtualMachineStateResource{}
var _ resource.ResourceWithImportState = &VirtualMachineStateResource{}

func NewVirtualMachineStateResource() resource.Resource {
	return &VirtualMachineStateResource{}
}

// VirtualMachineStateResource defines the resource implementation.
type VirtualMachineStateResource struct {
	provider *models.ParallelsProviderModel
}

func (r *VirtualMachineStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_state"
}

func (r *VirtualMachineStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = virtualMachineStateResourceSchema
}

func (r *VirtualMachineStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*models.ParallelsProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *models.ParallelsProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.provider = data
}

func (r *VirtualMachineStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}
	hostConfig := apiclient.HostConfig{
		Host:          data.Host.ValueString(),
		License:       r.provider.License.ValueString(),
		Authorization: data.Authenticator,
	}

	vm, diag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	result, diag := apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), r.GetOpState(data.Operation.ValueString()))
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if !result {
		resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
		return
	}

	data.Operation = types.StringValue(data.Operation.ValueString())
	data.CurrentState = types.StringValue(vm.State)

	tflog.Trace(ctx, "virtual machine "+vm.Name+" state changed to "+data.Operation.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:          data.Host.ValueString(),
		License:       r.provider.License.ValueString(),
		Authorization: data.Authenticator,
	}

	vm, diag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.CurrentState = types.StringValue(vm.State)

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	if data.ID.IsNull() {
		resp.Diagnostics.AddError("Id is required", "Id is required")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:          data.Host.ValueString(),
		License:       r.provider.License.ValueString(),
		Authorization: data.Authenticator,
	}

	vm, diag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// var proposedState string
	// switch strings.ToLower(data.Operation.ValueString()) {
	// case "start":
	// 	proposedState = "running"
	// case "stop":
	// 	proposedState = "stopped"
	// case "suspend":
	// 	proposedState = "suspended"
	// case "pause":
	// 	proposedState = "paused"
	// case "resume":
	// 	proposedState = "running"
	// case "restart":
	// 	proposedState = "running"
	// default:
	// 	resp.Diagnostics.AddError("invalid desired_state", "invalid desired_state")
	// 	return
	// }

	result, diag := apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), r.GetOpState(data.Operation.ValueString()))
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if !result {
		resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
		return
	}

	data.Operation = types.StringValue(data.Operation.ValueString())
	data.CurrentState = types.StringValue(vm.State)

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VirtualMachineStateResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VirtualMachineStateResource) GetOpState(value string) apiclient.MachineStateOp {
	switch strings.ToLower(value) {
	case "start":
		return apiclient.MachineStateOpStart
	case "stop":
		return apiclient.MachineStateOpStop
	case "suspend":
		return apiclient.MachineStateOpSuspend
	case "pause":
		return apiclient.MachineStateOpPause
	case "resume":
		return apiclient.MachineStateOpResume
	case "restart":
		return apiclient.MachineStateOpRestart
	default:
		return ""
	}
}
