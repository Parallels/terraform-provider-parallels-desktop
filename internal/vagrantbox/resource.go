package vagrantbox

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/common"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/telemetry"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &VagrantBoxResource{}
	_ resource.ResourceWithImportState = &VagrantBoxResource{}
)

func NewVagrantBoxResource() resource.Resource {
	return &VagrantBoxResource{}
}

// VagrantBoxResource defines the resource implementation.
type VagrantBoxResource struct {
	provider *models.ParallelsProviderModel
}

func (r *VagrantBoxResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vagrant_box"
}

func (r *VagrantBoxResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = getSchema(ctx)
}

func (r *VagrantBoxResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VagrantBoxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VagrantBoxResourceModel

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

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

	// selecting if this is a standalone host or an orchestrator
	isOrchestrator := false
	var host string
	if data.Orchestrator.ValueString() != "" {
		isOrchestrator = true
		host = data.Orchestrator.ValueString()
	} else {
		host = data.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		IsOrchestrator:       isOrchestrator,
		Host:                 data.Host.ValueString(),
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	if !isOrchestrator {
		// before creating, if we have enough data we will be checking if we have enough resources
		// in the current host if it is not an orchestrator, in that case it will be the orchestrator
		// job to check if we have enough resources
		if data.Specs != nil && !isOrchestrator {
			if diags := common.CheckIfEnoughSpecs(ctx, hostConfig, data.Specs, ""); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
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
		VagrantBox: &apimodels.CreateVagrantVmRequest{
			Box:                   data.BoxName.ValueString(),
			Version:               data.BoxVersion.ValueString(),
			VagrantFilePath:       data.VagrantFilePath.ValueString(),
			CustomVagrantConfig:   data.CustomVagrantConfig.ValueString(),
			CustomParallelsConfig: data.CustomParallelsConfig.ValueString(),
		},
	}

	if data.Owner.ValueString() != "" {
		createVmRequest.Owner = data.Owner.ValueString()
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

	// stopping the machine as it might need some operations where the machine needs to be stopped
	// add anything here in sequence that needs to be done before the machine is started
	// so we do not loose time waiting for the machine to stop
	stoppedVm, diag := common.EnsureMachineStopped(ctx, hostConfig, createdVM)
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
	}

	// Applying the Specs block
	if data.Specs != nil {
		if diags := common.SpecsBlockOnCreate(ctx, hostConfig, stoppedVm, data.Specs); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}
	}

	// Configuring the machine if there is any configuration
	if diag := common.VmConfigBlockOnCreate(ctx, hostConfig, stoppedVm, data.Config); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
		return
	}

	// Applying any prlctl commands
	if diag := common.PrlCtlBlockOnCreate(ctx, hostConfig, stoppedVm, data.PrlCtl); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
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

	// Starting the vm if requested
	if data.RunAfterCreate.ValueBool() {
		if _, diag := common.EnsureMachineRunning(ctx, hostConfig, stoppedVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}

		_, diag := apiclient.GetVm(ctx, hostConfig, response.ID)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	data.OsType = types.StringValue(createdVM.OS)
	if data.OnDestroyScript != nil {
		for _, script := range data.OnDestroyScript {
			elements := make([]attr.Value, 0)
			result := postprocessorscript.PostProcessorScriptRunResult{
				ExitCode: types.StringValue("0"),
				Stdout:   types.StringValue(""),
				Stderr:   types.StringValue(""),
				Script:   types.StringValue(""),
			}
			mappedObject, diag := result.MapObject(ctx)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			elements = append(elements, mappedObject)
			listValue, diag := types.ListValue(result.ElementType(ctx), elements)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			script.Result = listValue
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

func (r *VagrantBoxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VagrantBoxResourceModel

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

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

	// selecting if this is a standalone host or an orchestrator
	isOrchestrator := false
	var host string
	if data.Orchestrator.ValueString() != "" {
		isOrchestrator = true
		host = data.Orchestrator.ValueString()
	} else {
		host = data.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		IsOrchestrator:       isOrchestrator,
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

func (r *VagrantBoxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VagrantBoxResourceModel
	var currentData VagrantBoxResourceModel

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

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

	// selecting if this is a standalone host or an orchestrator
	isOrchestrator := false
	var host string
	if data.Orchestrator.ValueString() != "" {
		isOrchestrator = true
		host = data.Orchestrator.ValueString()
	} else {
		host = data.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		IsOrchestrator:       isOrchestrator,
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

	nameChanges := apimodels.NewVmConfigRequest(vm.User)
	currentState := vm.State
	needsRestart := false
	requireShutdown := false

	// Name is not the same, we will need to rename the machine
	if vm.Name != data.Name.ValueString() {
		op := apimodels.NewVmConfigRequestOperation(nameChanges)
		op.WithGroup("machine")
		op.WithOperation("rename")
		op.WithValue(data.Name.ValueString())
		op.Append()
	}

	configChanges := common.VmConfigBlockHasChanges(ctx, hostConfig, vm, data.Config, currentData.Config)
	specsChanges := common.SpecsBlockHasChanges(ctx, hostConfig, vm, data.Specs, currentData.Specs)
	prlctlChanges := common.PrlCtlBlockHasChanges(ctx, hostConfig, vm, data.PrlCtl, currentData.PrlCtl)
	postProcessorScriptChanges := common.PostProcessorHasChanges(ctx, data.PostProcessorScripts, currentData.PostProcessorScripts)
	if specsChanges || configChanges || prlctlChanges || nameChanges.HasChanges() {
		requireShutdown = true
	}

	if requireShutdown && vm.State != "stopped" {
		if data.ForceChanges.ValueBool() {
			if newVm, stopDiag := common.EnsureMachineStopped(ctx, hostConfig, vm); stopDiag.HasError() {
				resp.Diagnostics.Append(stopDiag...)
				return
			} else {
				vm = newVm
			}

			needsRestart = true
		} else {
			resp.Diagnostics.AddError("vm must be stopped before updating", "Virtual Machine "+vm.Name+" must be stopped before updating, currently "+vm.State)
			return
		}
	}

	// Applying the Specs block
	if specsChanges {
		if diags := common.SpecsBlockOnUpdate(ctx, hostConfig, vm, data.Specs, currentData.Specs); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Configuring the machine if there is any configuration
	if configChanges {
		if diag := common.VmConfigBlockOnUpdate(ctx, hostConfig, vm, data.Config, currentData.Config); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Applying any prlctl commands
	if prlctlChanges {
		if diag := common.PrlCtlBlockOnUpdate(ctx, hostConfig, vm, data.PrlCtl); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Restarting the machine if needed
	if needsRestart || (vm.State == "stopped" && currentState == "running") {
		if newVm, startDiags := common.EnsureMachineRunning(ctx, hostConfig, vm); startDiags.HasError() {
			resp.Diagnostics.Append(startDiags...)
			return
		} else {
			vm = newVm
		}
	}

	// Processing shared folders
	if diag := common.SharedFoldersBlockOnUpdate(ctx, hostConfig, vm, data.SharedFolder, currentData.SharedFolder); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// Running post processor changes
	if postProcessorScriptChanges {
		if diag := common.RunPostProcessorScript(ctx, hostConfig, vm, data.PostProcessorScripts); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	data.ID = types.StringValue(vm.ID)
	if data.OnDestroyScript != nil {
		for _, script := range data.OnDestroyScript {
			elements := make([]attr.Value, 0)
			result := postprocessorscript.PostProcessorScriptRunResult{
				ExitCode: types.StringValue("0"),
				Stdout:   types.StringValue(""),
				Stderr:   types.StringValue(""),
				Script:   types.StringValue(""),
			}
			mappedObject, diag := result.MapObject(ctx)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			elements = append(elements, mappedObject)
			listValue, diag := types.ListValue(result.ElementType(ctx), elements)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			script.Result = listValue
		}
	}

	tflog.Info(ctx, "Updated vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VagrantBoxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VagrantBoxResourceModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeDestroy,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(telemetryEvent)

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

	// selecting if this is a standalone host or an orchestrator
	isOrchestrator := false
	var host string
	if data.Orchestrator.ValueString() != "" {
		isOrchestrator = true
		host = data.Orchestrator.ValueString()
	} else {
		host = data.Host.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddError("host cannot be empty", "Host cannot be null")
		return
	}

	hostConfig := apiclient.HostConfig{
		IsOrchestrator:       isOrchestrator,
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

	// Running cleanup script if any
	if data.OnDestroyScript != nil {
		if diag := common.RunPostProcessorScript(ctx, hostConfig, vm, data.OnDestroyScript); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Stopping the machine
	if _, stopDiag := common.EnsureMachineStopped(ctx, hostConfig, vm); stopDiag.HasError() {
		resp.Diagnostics.Append(stopDiag...)
		return
	}

	deleteDiag := apiclient.DeleteVm(ctx, hostConfig, vm.ID)
	if deleteDiag.HasError() {
		resp.Diagnostics.Append(deleteDiag...)
		return
	}

	retryCount := 0
	for {
		vm, diag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			break
		}

		if vm == nil {
			break
		}

		retryCount += 1
		if retryCount >= 10 {
			resp.Diagnostics.AddError("error deleting vm", "error deleting vm")
			return
		}

		time.Sleep(10 * time.Second)
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VagrantBoxResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
