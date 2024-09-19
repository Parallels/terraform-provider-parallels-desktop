package packertemplate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/common"
	"terraform-provider-parallels-desktop/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &PackerTemplateVirtualMachineResource{}
	_ resource.ResourceWithImportState = &PackerTemplateVirtualMachineResource{}
)

func NewPackerTemplateVirtualMachineResource() resource.Resource {
	return &PackerTemplateVirtualMachineResource{}
}

// PackerTemplateVirtualMachineResource defines the resource implementation.
type PackerTemplateVirtualMachineResource struct {
	provider *models.ParallelsProviderModel
}

func (r *PackerTemplateVirtualMachineResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_packer_template"
}

func (r *PackerTemplateVirtualMachineResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = getSchema(ctx)
}

func (r *PackerTemplateVirtualMachineResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PackerTemplateVirtualMachineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PackerVirtualMachineResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	vm, diag := apiclient.GetVms(ctx, hostConfig, "Name", data.Name.String())
	if diag.HasError() {
		diag.Append(diag...)
		return
	}

	if len(vm) > 0 {
		resp.Diagnostics.AddError("Vm already exists", "The vm "+data.Name.ValueString()+" already exists")
		return
	}

	createVmRequest := apimodels.CreateVmRequest{
		Name: data.Name.ValueString(),
		PackerTemplate: &apimodels.CreatePackerVmRequest{
			Template: data.Template.ValueString(),
		},
	}

	if data.Owner.ValueString() != "" {
		createVmRequest.Owner = data.Owner.ValueString()
	}

	if data.Specs != nil {
		if data.Specs.CpuCount.ValueString() != "" {
			createVmRequest.PackerTemplate.Cpu = data.Specs.CpuCount.ValueString()
		}
		if data.Specs.MemorySize.ValueString() != "" {
			createVmRequest.PackerTemplate.Memory = data.Specs.MemorySize.ValueString()
		}
		if data.Specs.DiskSize.ValueString() != "" {
			createVmRequest.PackerTemplate.Disk = data.Specs.DiskSize.ValueString()
		}
	}

	response, diag := apiclient.CreateVm(ctx, hostConfig, createVmRequest)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	data.ID = types.StringValue(response.ID)
	tflog.Info(ctx, "Created vm with id "+data.ID.ValueString())

	createdVM, diag := apiclient.GetVm(ctx, hostConfig, response.ID)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	if createdVM == nil {
		resp.Diagnostics.AddError("vm was not found", "vm was not found")
		return
	}

	// Processing shared folders
	if diag := common.SharedFoldersBlockOnCreate(ctx, hostConfig, createdVM, data.SharedFolder); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
		return
	}

	// Running the post processor scripts
	if diag := common.RunPostProcessorScript(ctx, hostConfig, createdVM, data.PostProcessorScripts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
		return
	}

	data.OsType = types.StringValue(createdVM.OS)
	if data.RunAfterCreate.ValueBool() {
		isRunning, diag := apiclient.SetMachineState(ctx, hostConfig, createdVM.ID, apiclient.MachineStateOpStart)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
		}
		if !isRunning {
			resp.Diagnostics.AddError("error starting vm", "error starting vm")
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PackerVirtualMachineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
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

	data.Name = types.StringValue(vm.Name)

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PackerVirtualMachineResourceModel
	var currentData PackerVirtualMachineResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	vm, diag := apiclient.GetVm(ctx, hostConfig, currentData.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	changes := apimodels.NewVmConfigRequest(vm.User)

	// Name is not the same, we will need to rename the machine
	if vm.Name != data.Name.ValueString() {
		op := apimodels.NewVmConfigRequestOperation(changes)
		op.WithGroup("machine")
		op.WithOperation("rename")
		op.WithValue(data.Name.ValueString())
		op.Append()
	}

	if data.Specs != nil {
		if data.Specs.CpuCount.ValueString() != fmt.Sprintf("%v", vm.Hardware.CPU.Cpus) {
			updateValue := data.Specs.CpuCount.ValueString()
			if updateValue == "" {
				updateValue = "2"
			}

			op := apimodels.NewVmConfigRequestOperation(changes)
			op.WithGroup("cpu")
			op.WithOperation("set")
			op.WithValue(updateValue)
			if currentData.Specs == nil || currentData.Specs.CpuCount.ValueString() != updateValue {
				op.Append()
			}
		}

		if data.Specs.MemorySize.ValueString() != "" && data.Specs.MemorySize.ValueString() != strings.ReplaceAll(vm.Hardware.Memory.Size, "Mb", "") {
			updateValue := data.Specs.CpuCount.ValueString()
			if updateValue == "" {
				updateValue = "2048"
			}

			op := apimodels.NewVmConfigRequestOperation(changes)
			op.WithGroup("memory")
			op.WithOperation("set")
			op.WithValue(updateValue)
			if currentData.Specs == nil || currentData.Specs.MemorySize.ValueString() != updateValue {
				op.Append()
			}
		}
	}

	needsRestart := false
	if changes.HasChanges() {
		if vm.State != "stopped" {
			if data.ForceChanges.ValueBool() {
				result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, vm.ID, apiclient.MachineStateOpStop)
				if stateDiag.HasError() {
					resp.Diagnostics.Append(stateDiag...)
					return
				}
				if !result {
					resp.Diagnostics.AddError("error stopping vm", "error stopping vm")
					return
				}
				needsRestart = true
			} else {
				resp.Diagnostics.AddError("vm must be stopped before updating", "Virtual Machine "+vm.Name+" must be stopped before updating, currently "+vm.State)
				return
			}
		}

		tflog.Info(ctx, "Updating vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString()+" with changes: "+changes.String())

		if _, diag := apiclient.ConfigureMachine(ctx, hostConfig, vm.ID, changes); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}

		if needsRestart {
			result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, vm.ID, apiclient.MachineStateOpStart)
			if stateDiag.HasError() {
				resp.Diagnostics.Append(stateDiag...)
				return
			}
			if !result {
				resp.Diagnostics.AddError("error starting vm", "error starting vm")
				return
			}

			// Sleep for a minute
			time.Sleep(time.Minute)
		}
	}

	// Processing shared folders
	if diag := common.SharedFoldersBlockOnUpdate(ctx, hostConfig, vm, data.SharedFolder, currentData.SharedFolder); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// Running post processor changes
	if diag := common.RunPostProcessorScript(ctx, hostConfig, vm, data.PostProcessorScripts); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	data.ID = types.StringValue(vm.ID)
	data.OsType = types.StringValue(vm.OS)

	tflog.Info(ctx, "Updated vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PackerVirtualMachineResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Setting the default timeout
	createTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	if data.Host.ValueString() == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	vm, diag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// Nothing to do, machine does not exist
	if vm == nil {
		resp.Diagnostics.Append(req.State.Set(ctx, &data)...)
		return
	}

	if vm.State != "stopped" {
		result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, vm.ID, apiclient.MachineStateOpStop)
		if stateDiag.HasError() {
			resp.Diagnostics.Append(stateDiag...)
			return
		}
		if !result {
			resp.Diagnostics.AddError("error stopping vm", "error stopping vm")
			return
		}
	}

	deleteDiag := apiclient.DeleteVm(ctx, hostConfig, vm.ID)
	if deleteDiag.HasError() {
		resp.Diagnostics.Append(deleteDiag...)
		return
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *PackerTemplateVirtualMachineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
