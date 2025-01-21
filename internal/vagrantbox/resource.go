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
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/telemetry"
	resource_models "terraform-provider-parallels-desktop/internal/vagrantbox/models"
	"terraform-provider-parallels-desktop/internal/vagrantbox/schemas"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	resp.Schema = schemas.GetResourceSchemaV1(ctx)
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
	var data resource_models.VagrantBoxResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

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
		if data.Specs != nil {
			if diags := common.CheckIfEnoughSpecs(ctx, hostConfig, data.Specs, ""); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	vm, getVmDiag := apiclient.GetVms(ctx, hostConfig, "Name", data.Name.String())
	if getVmDiag.HasError() {
		resp.Diagnostics.Append(getVmDiag...)
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

	response, createVmDiag := apiclient.CreateVm(ctx, hostConfig, createVmRequest)
	if createVmDiag.HasError() {
		resp.Diagnostics.Append(createVmDiag...)
		return
	}

	data.ID = types.StringValue(response.ID)
	tflog.Info(ctx, "Created vm with id "+data.ID.ValueString())

	createdVM, getVmDiag := apiclient.GetVm(ctx, hostConfig, response.ID)
	if getVmDiag.HasError() {
		resp.Diagnostics.Append(getVmDiag...)
		return
	}

	if createdVM == nil {
		resp.Diagnostics.AddError("vm was not found", "vm was not found")
		return
	}

	// stopping the machine as it might need some operations where the machine needs to be stopped
	// add anything here in sequence that needs to be done before the machine is started
	// so we do not loose time waiting for the machine to stop
	stoppedVm, stoppedVmDiag := common.EnsureMachineStopped(ctx, hostConfig, createdVM)
	if stoppedVmDiag.HasError() {
		resp.Diagnostics.Append(stoppedVmDiag...)
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

	if len(data.ReverseProxyHosts) > 0 {
		rpHostConfig := hostConfig
		rpHostConfig.HostId = stoppedVm.HostId
		rpHosts, updateDiag := updateReverseProxyHostsTarget(ctx, &data, rpHostConfig, stoppedVm)
		if updateDiag.HasError() {
			resp.Diagnostics.Append(updateDiag...)
			return
		}

		result, createDiag := reverseproxy.Create(ctx, rpHostConfig, rpHosts)
		if createDiag.HasError() {
			resp.Diagnostics.Append(createDiag...)

			if diag := reverseproxy.Delete(ctx, rpHostConfig, rpHosts); diag.HasError() {
				tflog.Error(ctx, "Error deleting reverse proxy hosts")
			}

			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, rpHostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(ctx, rpHostConfig, stoppedVm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}

				apiclient.DeleteVm(ctx, rpHostConfig, data.ID.ValueString())
			}
			return
		}

		for i := range result {
			data.ReverseProxyHosts[i].ID = result[i].ID
		}
	}

	// Starting the vm by default, otherwise we will stop the VM from being created
	if data.RunAfterCreate.ValueBool() || data.KeepRunning.ValueBool() || (data.RunAfterCreate.IsUnknown() && data.KeepRunning.IsUnknown()) {
		if _, diag := common.EnsureMachineRunning(ctx, hostConfig, stoppedVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(ctx, hostConfig, stoppedVm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}

		_, diag := apiclient.GetVm(ctx, hostConfig, response.ID)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	} else {
		// If we are not starting the machine, we will stop it
		if _, diag := common.EnsureMachineStopped(ctx, hostConfig, stoppedVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(ctx, hostConfig, stoppedVm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}
	}

	externalIp := ""
	internalIp := ""
	retryAttempts := 10
	var refreshVm *apimodels.VirtualMachine
	var refreshDiag diag.Diagnostics
	for {
		refreshVm, refreshDiag = apiclient.GetVm(ctx, hostConfig, response.ID)
		if !refreshDiag.HasError() {
			externalIp = refreshVm.HostExternalIpAddress
			internalIp = refreshVm.InternalIpAddress
		}

		if refreshVm.State != "running" {
			externalIp = refreshVm.HostExternalIpAddress
			internalIp = "-"
			break
		}

		if internalIp != "" {
			time.Sleep(5 * time.Second)
			break
		}
		if retryAttempts == 0 {
			internalIp = "-"
			break
		}
		retryAttempts--
	}

	data.ExternalIp = types.StringValue(externalIp)
	data.InternalIp = types.StringValue(internalIp)

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
	var data resource_models.VagrantBoxResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

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
	var data resource_models.VagrantBoxResourceModelV1
	var currentData resource_models.VagrantBoxResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVagrant, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

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

	vm, getVmDiag := apiclient.GetVm(ctx, hostConfig, currentData.ID.ValueString())
	if getVmDiag.HasError() {
		resp.Diagnostics.Append(getVmDiag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	currentVmState := vm.State

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

	if reverseproxy.ReverseProxyHostsDiff(data.ReverseProxyHosts, currentData.ReverseProxyHosts) {
		copyCurrentRpHosts := reverseproxy.CopyReverseProxyHosts(currentData.ReverseProxyHosts)
		copyRpHosts := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)

		results, updateDiag := reverseproxy.Update(ctx, hostConfig, copyCurrentRpHosts, copyRpHosts)
		if updateDiag.HasError() {
			resp.Diagnostics.Append(updateDiag...)
			revertResults, _ := reverseproxy.Revert(ctx, hostConfig, copyCurrentRpHosts, copyRpHosts)
			for i := range revertResults {
				data.ReverseProxyHosts[i].ID = revertResults[i].ID
			}
			return
		}

		for i := range results {
			data.ReverseProxyHosts[i].ID = results[i].ID
		}
	} else {
		for i := range currentData.ReverseProxyHosts {
			data.ReverseProxyHosts[i].ID = currentData.ReverseProxyHosts[i].ID
		}
	}

	// Starting the vm by default, otherwise we will stop the VM from being created
	if data.RunAfterCreate.ValueBool() || data.KeepRunning.ValueBool() || (data.RunAfterCreate.IsUnknown() && data.KeepRunning.IsUnknown()) {
		if _, diag := common.EnsureMachineRunning(ctx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(ctx, hostConfig, vm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}

		_, diag := apiclient.GetVm(ctx, hostConfig, vm.ID)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	} else {
		// If we are not starting the machine, we will stop it
		if _, diag := common.EnsureMachineStopped(ctx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(ctx, hostConfig, vm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
				apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
			}
			return
		}
	}

	externalIp := ""
	internalIp := ""
	retryAttempts := 10
	var refreshVm *apimodels.VirtualMachine
	var refreshDiag diag.Diagnostics
	for {
		refreshVm, refreshDiag = apiclient.GetVm(ctx, hostConfig, vm.ID)
		if !refreshDiag.HasError() {
			externalIp = refreshVm.HostExternalIpAddress
			internalIp = refreshVm.InternalIpAddress
		}

		if refreshVm.State != "running" {
			externalIp = refreshVm.HostExternalIpAddress
			internalIp = "-"
			break
		}

		if internalIp != "" {
			time.Sleep(5 * time.Second)
			break
		}
		if retryAttempts == 0 {
			internalIp = "-"
			break
		}
		retryAttempts--
	}

	data.ExternalIp = types.StringValue(externalIp)
	data.InternalIp = types.StringValue(internalIp)

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

	if currentVmState != refreshVm.State {
		// If the vm state is desync we nee to set it right
		switch currentVmState {
		case "running":
			if refreshVm.State == "stopped" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
			if refreshVm.State == "paused" || refreshVm.State == "suspended" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
		case "stopped":
			if refreshVm.State == "running" || refreshVm.State == "paused" || refreshVm.State == "suspended" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			}
		case "paused":
			if refreshVm.State == "running" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
			if refreshVm.State == "stopped" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
		case "suspended":
			if refreshVm.State == "running" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
			if refreshVm.State == "stopped" {
				apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
		}
	}

	tflog.Info(ctx, "Updated vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VagrantBoxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_models.VagrantBoxResourceModelV1
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
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

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

func updateReverseProxyHostsTarget(ctx context.Context, data *resource_models.VagrantBoxResourceModelV1, hostConfig apiclient.HostConfig, targetVm *apimodels.VirtualMachine) ([]reverseproxy.ReverseProxyHost, diag.Diagnostics) {
	resultDiagnostic := diag.Diagnostics{}
	var refreshedVm *apimodels.VirtualMachine
	var rpDiag diag.Diagnostics
	refreshedVm, rpDiag = common.EnsureMachineHasInternalIp(ctx, hostConfig, targetVm)
	if rpDiag.HasError() {
		resultDiagnostic.Append(rpDiag...)
		if data.ID.ValueString() != "" {
			// If we have an ID, we need to delete the machine
			apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			if _, diag := common.EnsureMachineStopped(ctx, hostConfig, refreshedVm); diag.HasError() {
				return nil, diag
			}
			apiclient.DeleteVm(ctx, hostConfig, data.ID.ValueString())
		}
		return nil, resultDiagnostic
	}

	modifiedHosts := make([]reverseproxy.ReverseProxyHost, len(data.ReverseProxyHosts))
	for i := range data.ReverseProxyHosts {
		host := reverseproxy.ReverseProxyHost{}
		host.Host = data.ReverseProxyHosts[i].Host
		host.Port = data.ReverseProxyHosts[i].Port
		internalIp := refreshedVm.InternalIpAddress
		emptyString := ""

		if data.ReverseProxyHosts[i].Cors != nil {
			host.Cors = &reverseproxy.ReverseProxyCors{}
			host.Cors.AllowedOrigins = data.ReverseProxyHosts[i].Cors.AllowedOrigins
			host.Cors.AllowedMethods = data.ReverseProxyHosts[i].Cors.AllowedMethods
			host.Cors.AllowedHeaders = data.ReverseProxyHosts[i].Cors.AllowedHeaders
			host.Cors.Enabled = data.ReverseProxyHosts[i].Cors.Enabled
		}
		if data.ReverseProxyHosts[i].Tls != nil {
			host.Tls = &reverseproxy.ReverseProxyTls{}
			host.Tls.Certificate = data.ReverseProxyHosts[i].Tls.Certificate
			host.Tls.PrivateKey = data.ReverseProxyHosts[i].Tls.PrivateKey
			host.Tls.Enabled = data.ReverseProxyHosts[i].Tls.Enabled
		}
		if data.ReverseProxyHosts[i].TcpRoute != nil {
			host.TcpRoute = &reverseproxy.ReverseProxyHostTcpRoute{}
			host.TcpRoute.TargetPort = data.ReverseProxyHosts[i].TcpRoute.TargetPort
			host.TcpRoute.TargetHost = types.StringValue(internalIp)
			host.TcpRoute.TargetVmId = types.StringValue(emptyString)
		}

		if len(data.ReverseProxyHosts[i].HttpRoute) > 0 {
			host.HttpRoute = make([]*reverseproxy.ReverseProxyHttpRoute, len(data.ReverseProxyHosts[i].HttpRoute))
			for j := range modifiedHosts[i].HttpRoute {
				httpRoute := reverseproxy.ReverseProxyHttpRoute{}
				httpRoute.Path = data.ReverseProxyHosts[i].HttpRoute[j].Path
				httpRoute.TargetHost = types.StringValue(internalIp)
				httpRoute.TargetPort = data.ReverseProxyHosts[i].HttpRoute[j].TargetPort
				httpRoute.TargetVmId = types.StringValue(emptyString)
				httpRoute.Pattern = data.ReverseProxyHosts[i].HttpRoute[j].Pattern
				httpRoute.Schema = data.ReverseProxyHosts[i].HttpRoute[j].Schema
				httpRoute.RequestHeaders = data.ReverseProxyHosts[i].HttpRoute[j].RequestHeaders
				httpRoute.ResponseHeaders = data.ReverseProxyHosts[i].HttpRoute[j].ResponseHeaders
				host.HttpRoute[j] = &httpRoute
			}
		}

		modifiedHosts[i] = host
	}

	return modifiedHosts, resultDiagnostic
}
