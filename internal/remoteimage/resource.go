package remoteimage

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/common"
	common_models "terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/remoteimage/models"
	"terraform-provider-parallels-desktop/internal/remoteimage/schemas"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/telemetry"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &RemoteVmResource{}
	_ resource.ResourceWithImportState = &RemoteVmResource{}
)

func NewRemoteVmResource() resource.Resource {
	return &RemoteVmResource{}
}

// RemoteVmResource defines the resource implementation.
type RemoteVmResource struct {
	provider *common_models.ParallelsProviderModel
}

func (r *RemoteVmResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote_vm"
}

func (r *RemoteVmResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schemas.GetRemoteImageSchemaV1(ctx)
}

func (r *RemoteVmResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*common_models.ParallelsProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common_modesl.ParallelsProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.provider = data
}

func (r *RemoteVmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.RemoteVmResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeCreate,
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
		Host:                 host,
		IsOrchestrator:       isOrchestrator,
		License:              r.provider.License.ValueString(),
		Authorization:        data.Authenticator,
		DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool(),
	}

	catalogHostConfig, err := common.ParseHostConnectionString(data.CatalogConnection.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error parsing host connection string", err.Error())
		return
	}

	catalogManifest, catalogManifestDiag := apiclient.GetCatalogManifest(ctx, *catalogHostConfig, data.CatalogId.ValueString(), data.Version.ValueString(), data.Architecture.ValueString())
	if catalogManifestDiag.HasError() {
		resp.Diagnostics.AddError("Catalog Not Found", fmt.Sprintf("Catalog %s was not found on %s", data.CatalogId.ValueString(), catalogHostConfig.Host))
		return
	}
	// the catalog manifest is nil, we will add an error to the diagnostics
	if catalogManifest == nil {
		resp.Diagnostics.AddError("Catalog Not Found", fmt.Sprintf("Catalog %s was not found on %s", data.CatalogId.ValueString(), catalogHostConfig.Host))
		return
	}

	// Checking if the VM already exists in the host
	vm, vmDiag := apiclient.GetVms(ctx, hostConfig, "Name", data.Name.String())
	if vmDiag.HasError() {
		resp.Diagnostics.Append(vmDiag...)
		return
	}

	// before creating, if we have enough data we will be checking if we have enough resources
	// in the current host
	// in the current host we will skip this if the host is an orchestrator
	if data.Specs != nil {
		architecture := data.Architecture.ValueString()
		if architecture == "" {
			architecture = catalogManifest.Architecture
		}

		if specsDiag := common.CheckIfEnoughSpecs(ctx, hostConfig, data.Specs, architecture); specsDiag.HasError() {
			resp.Diagnostics.Append(specsDiag...)
			return
		}
	}

	if len(vm) > 0 {
		resp.Diagnostics.AddError("Vm already exists", "The vm "+data.Name.ValueString()+" already exists")
		return
	}

	version := catalogManifest.Version
	architecture := catalogManifest.Architecture
	if data.Version.ValueString() != "" {
		version = data.Version.ValueString()
	}
	if data.Architecture.ValueString() != "" {
		architecture = data.Architecture.ValueString()
	}

	createMachineRequest := apimodels.CreateVmRequest{
		Name:         data.Name.ValueString(),
		Architecture: architecture,
		CatalogManifest: &apimodels.CreateCatalogManifestRequest{
			MachineName:    data.Name.ValueString(),
			CatalogId:      data.CatalogId.ValueString(),
			Version:        version,
			Architecture:   architecture,
			Connection:     data.CatalogConnection.ValueString(),
			StartAfterPull: data.RunAfterCreate.ValueBool(),
			Path:           data.Path.ValueString(),
		},
	}

	if data.Owner.ValueString() != "" {
		createMachineRequest.Owner = data.Owner.ValueString()
	}

	response, vmDiag := apiclient.CreateVm(ctx, hostConfig, createMachineRequest)
	if vmDiag.HasError() {
		resp.Diagnostics.Append(vmDiag...)
		return
	}

	data.ID = types.StringValue(response.ID)
	tflog.Info(ctx, "Created vm with id "+data.ID.ValueString())

	createdVM, vmDiag := apiclient.GetVm(ctx, hostConfig, response.ID)
	if vmDiag.HasError() {
		resp.Diagnostics.Append(vmDiag...)
		return
	}

	hostConfig.HostId = createdVM.HostId

	// stopping the machine as it might need some operations where the machine needs to be stopped
	// add anything here in sequence that needs to be done before the machine is started
	// so we do not loose time waiting for the machine to stop
	stoppedVm, vmDiag := common.EnsureMachineStopped(ctx, hostConfig, createdVM)
	if vmDiag.HasError() {
		resp.Diagnostics.Append(vmDiag...)
	}

	// Applying the Specs block
	if data.Specs != nil {
		if diags := common.SpecsBlockOnCreate(ctx, hostConfig, stoppedVm, data.Specs); diags.HasError() {
			resp.Diagnostics.Append(diags...)
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

	// Configuring the machine if there is any configuration
	if vmBlockDiag := common.VmConfigBlockOnCreate(ctx, hostConfig, stoppedVm, data.Config); vmBlockDiag.HasError() {
		resp.Diagnostics.Append(vmBlockDiag...)
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

	// Applying any prlctl commands
	if prlctlDiag := common.PrlCtlBlockOnCreate(ctx, hostConfig, stoppedVm, data.PrlCtl); prlctlDiag.HasError() {
		resp.Diagnostics.Append(prlctlDiag...)
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

	// Processing shared folders
	if sharedFolderDiag := common.SharedFoldersBlockOnCreate(ctx, hostConfig, stoppedVm, data.SharedFolder); sharedFolderDiag.HasError() {
		resp.Diagnostics.Append(sharedFolderDiag...)
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

	// Running the post processor scripts
	if postProcessDiag := common.RunPostProcessorScript(ctx, hostConfig, stoppedVm, data.PostProcessorScripts); postProcessDiag.HasError() {
		resp.Diagnostics.Append(postProcessDiag...)
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
	refreshVm, refreshDiag := apiclient.GetVm(ctx, hostConfig, response.ID)
	if refreshDiag.HasError() {
		resp.Diagnostics.Append(refreshDiag...)
		return
	} else {
		externalIp = refreshVm.HostExternalIpAddress
		internalIp = refreshVm.InternalIpAddress
	}

	data.OsType = types.StringValue(createdVM.OS)
	data.ExternalIp = types.StringValue(externalIp)
	data.InternalIp = types.StringValue(internalIp)
	data.OrchestratorHostId = types.StringValue(refreshVm.HostId)
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

	// Setting the state of the vm to running if that is required
	if (data.RunAfterCreate.ValueBool() || data.RunAfterCreate.IsUnknown() || data.RunAfterCreate.IsNull()) && (refreshVm.State == "stopped") {
		if _, diag := common.EnsureMachineRunning(ctx, hostConfig, refreshVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
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

func (r *RemoteVmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.RemoteVmResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeRead,
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
		Host:                 host,
		IsOrchestrator:       isOrchestrator,
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

func (r *RemoteVmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.RemoteVmResourceModelV1
	var currentData models.RemoteVmResourceModelV1

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeUpdate,
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
		Host:                 host,
		IsOrchestrator:       isOrchestrator,
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

	hostConfig.HostId = vm.HostId

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

	// Changing the name of the machine
	if nameChanges.HasChanges() {
		_, diag := apiclient.ConfigureMachine(ctx, hostConfig, vm.ID, nameChanges)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Applying the Specs block
	if specsChanges {
		// We need to stop the machine if it is not stopped
		if vm.State != "stopped" && !data.ForceChanges.ValueBool() {
			resp.Diagnostics.AddError("vm must be stopped before updating", "Virtual Machine "+vm.Name+" must be stopped before updating, currently "+vm.State)
			return
		}

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

	// Running the post processor scripts
	if postProcessorScriptChanges {
		if diag := common.RunPostProcessorScript(ctx, hostConfig, vm, data.PostProcessorScripts); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	} else {
		data.PostProcessorScripts = currentData.PostProcessorScripts
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
	refreshVm, refreshDiag := apiclient.GetVm(ctx, hostConfig, vm.ID)
	if refreshDiag.HasError() {
		resp.Diagnostics.Append(refreshDiag...)
		return
	} else {
		externalIp = refreshVm.HostExternalIpAddress
		internalIp = refreshVm.InternalIpAddress
	}

	data.ID = types.StringValue(vm.ID)
	data.OsType = types.StringValue(refreshVm.OS)
	data.ExternalIp = types.StringValue(externalIp)
	data.InternalIp = types.StringValue(internalIp)
	data.OrchestratorHostId = types.StringValue(refreshVm.HostId)

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

	if (data.RunAfterCreate.ValueBool() || data.RunAfterCreate.IsUnknown() || data.RunAfterCreate.IsNull()) && (vm.State == "stopped") {
		if _, diag := common.EnsureMachineRunning(ctx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	tflog.Info(ctx, "Updated vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteVmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.RemoteVmResourceModelV1
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeDestroy,
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
		Host:                 host,
		IsOrchestrator:       isOrchestrator,
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
	} else {
		hostConfig.HostId = vm.HostId
	}

	// Running cleanup script if any
	if data.OnDestroyScript != nil {
		_ = common.RunPostProcessorScript(ctx, hostConfig, vm, data.OnDestroyScript)
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

	if len(data.ReverseProxyHosts) > 0 {
		rpHostConfig := hostConfig
		rpHostConfig.HostId = vm.HostId
		rpHostsCopy := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)
		if diag := reverseproxy.Delete(ctx, rpHostConfig, rpHostsCopy); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	resp.Diagnostics.Append(req.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteVmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RemoteVmResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	v0Schema := schemas.GetRemoteImageSchemaV0(ctx)
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &v0Schema,
			StateUpgrader: UpgradeStateToV1,
		},
	}
}

func UpgradeStateToV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var priorStateData models.RemoteVmResourceModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	upgradedStateData := models.RemoteVmResourceModelV1{
		Authenticator:        priorStateData.Authenticator,
		Host:                 priorStateData.Host,
		Orchestrator:         priorStateData.Orchestrator,
		ID:                   priorStateData.ID,
		OsType:               priorStateData.OsType,
		ExternalIp:           types.StringUnknown(),
		InternalIp:           types.StringUnknown(),
		OrchestratorHostId:   types.StringUnknown(),
		CatalogId:            priorStateData.CatalogId,
		Version:              priorStateData.Version,
		Architecture:         priorStateData.Architecture,
		Name:                 priorStateData.Name,
		Owner:                priorStateData.Owner,
		CatalogConnection:    priorStateData.CatalogConnection,
		Path:                 priorStateData.Path,
		Specs:                priorStateData.Specs,
		PostProcessorScripts: priorStateData.PostProcessorScripts,
		OnDestroyScript:      priorStateData.OnDestroyScript,
		SharedFolder:         priorStateData.SharedFolder,
		Config:               priorStateData.Config,
		PrlCtl:               priorStateData.PrlCtl,
		RunAfterCreate:       priorStateData.RunAfterCreate,
		Timeouts:             priorStateData.Timeouts,
		ForceChanges:         priorStateData.ForceChanges,
		KeepRunning:          types.BoolValue(true),
		ReverseProxyHosts:    make([]*reverseproxy.ReverseProxyHost, 0),
	}

	println(fmt.Sprintf("Upgrading state from version %v", upgradedStateData))

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgradedStateData)...)
}

func updateReverseProxyHostsTarget(ctx context.Context, data *models.RemoteVmResourceModelV1, hostConfig apiclient.HostConfig, targetVm *apimodels.VirtualMachine) ([]reverseproxy.ReverseProxyHost, diag.Diagnostics) {
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
