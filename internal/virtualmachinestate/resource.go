package virtualmachinestate

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-parallels/internal/clientmodels"
	"terraform-provider-parallels/internal/constants"
	"terraform-provider-parallels/internal/helpers"

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
}

func (r *VirtualMachineStateResource) getVm(ctx context.Context, url string) (*clientmodels.VirtualMachine, error) {
	tflog.Info(ctx, "Getting VM from "+url)
	var result clientmodels.VirtualMachine

	caller := helpers.NewHttpCaller(ctx)
	_, err := caller.GetDataFromClient(url, nil, helpers.HttpCallerAuth{}, &result)
	if err != nil {
		return nil, err
	}

	if result.ID == "" {
		return nil, fmt.Errorf("VM not found")
	}

	return &result, nil
}

func (r *VirtualMachineStateResource) setState(ctx context.Context, data VirtualMachineStateResourceModel) (*clientmodels.VirtualMachine, error) {
	if data.Host.IsNull() {
		return nil, fmt.Errorf("Host is required")
	}

	if data.ID.IsNull() {
		return nil, fmt.Errorf("Id is required")
	}

	baseUrl := fmt.Sprintf("%s/%s/%s/%s",
		helpers.CleanUrlSuffixAndPrefix(data.Host.ValueString()),
		helpers.CleanUrlSuffixAndPrefix(constants.API_PREFIX),
		"machines", data.ID.ValueString())

	vm, err := r.getVm(ctx, baseUrl)
	if err != nil {
		return nil, err
	}
	if vm == nil {
		return nil, fmt.Errorf("machine not found")
	}

	var queryBaseUrl string
	switch strings.ToLower(data.Operation.ValueString()) {
	case "start":
		if vm.State != "stopped" {
			return nil, fmt.Errorf("machine is not stopped")
		}
		queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "start")
	case "stop":
		if vm.State != "running" {
			return nil, fmt.Errorf("machine is not running")
		}
		queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "stop")
	case "suspend":
		if vm.State != "running" {
			return nil, fmt.Errorf("machine is not running")
		}
		queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "suspend")
	case "pause":
		if vm.State != "running" {
			return nil, fmt.Errorf("machine is not running")
		}
		queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "pause")
	case "resume":
		if vm.State != "suspended" && vm.State != "paused" {
			return nil, fmt.Errorf("machine is not suspended or paused")
		}
		queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "resume")
	case "restart":
		if vm.State == "running" {
			queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "restart")
		} else {
			queryBaseUrl = fmt.Sprintf("%s/%s", baseUrl, "start")
		}
	default:
		return nil, fmt.Errorf("invalid desired_state")
	}

	var result clientmodels.VirtualMachineStateResponse
	caller := helpers.NewHttpCaller(ctx)
	_, err = caller.GetDataFromClient(queryBaseUrl, nil, helpers.HttpCallerAuth{}, &result)
	if err != nil {
		return nil, err
	}

	if result.Status != "Success" {
		return nil, fmt.Errorf("error changing the machine state")
	}

	vm, err = r.getVm(ctx, baseUrl)
	if err != nil {
		return nil, err
	}

	return vm, nil
}

func (r *VirtualMachineStateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm_state"
}

func (r *VirtualMachineStateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = virtualMachineStateResourceSchema
}

func (r *VirtualMachineStateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	return
}

func (r *VirtualMachineStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := r.setState(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("error changing the machine state", err.Error())
		return
	}

	data.Operation = types.StringValue(data.Operation.ValueString())
	data.CurrentState = types.StringValue(vm.State)

	tflog.Trace(ctx, "virtual machine "+data.ID.ValueString()+" state changed to "+data.Operation.ValueString()+" on host "+data.Host.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VirtualMachineStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VirtualMachineStateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	baseUrl := fmt.Sprintf("%s/%s/%s/%s",
		helpers.CleanUrlSuffixAndPrefix(data.Host.ValueString()),
		helpers.CleanUrlSuffixAndPrefix(constants.API_PREFIX),
		"machines", data.ID.ValueString())

	vm, err := r.getVm(ctx, baseUrl)

	if err != nil {
		resp.Diagnostics.AddError("error getting machine", err.Error())
		return
	}

	if vm == nil {
		resp.Diagnostics.AddError("machine not found", "machine not found")
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

	if data.Host.IsNull() {
		resp.Diagnostics.AddError("Host is required", "Host is required")
		return
	}

	if data.ID.IsNull() {
		resp.Diagnostics.AddError("Id is required", "Id is required")
		return
	}

	baseUrl := fmt.Sprintf("%s/%s/%s/%s",
		helpers.CleanUrlSuffixAndPrefix(data.Host.ValueString()),
		helpers.CleanUrlSuffixAndPrefix(constants.API_PREFIX),
		"machines", data.ID.ValueString())

	vm, err := r.getVm(ctx, baseUrl)
	if err != nil {
		resp.Diagnostics.AddError("error getting machine", err.Error())
		return
	}
	if vm == nil {
		resp.Diagnostics.AddError("machine not found", "machine not found")
		return
	}

	var proposedState string
	switch strings.ToLower(data.Operation.ValueString()) {
	case "start":
		proposedState = "running"
	case "stop":
		proposedState = "stopped"
	case "suspend":
		proposedState = "suspended"
	case "pause":
		proposedState = "paused"
	case "resume":
		proposedState = "running"
	case "restart":
		proposedState = "running"
	default:
		resp.Diagnostics.AddError("invalid desired_state", "invalid desired_state")
		return
	}

	if vm.State != proposedState {
		vm, err = r.setState(ctx, data)
		if err != nil {
			resp.Diagnostics.AddError("error changing the machine state", err.Error())
			return
		}

		data.CurrentState = types.StringValue(vm.State)
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	return
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
