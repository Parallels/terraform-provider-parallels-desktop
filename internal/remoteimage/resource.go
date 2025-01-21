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
	resp.Schema = schemas.GetRemoteImageSchemaV2(ctx)
}

func (r *RemoteVmResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*common_models.ParallelsProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common_models .ParallelsProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.provider = data
}

func (r *RemoteVmResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data models.RemoteVmResourceModelV2

	// Setting the default timeout
	contextTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(apiCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		apiCtx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(apiCtx, telemetryEvent)

	resp.Diagnostics.Append(req.Plan.Get(apiCtx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	catalogManifest, catalogManifestDiag := apiclient.GetCatalogManifest(apiCtx, *catalogHostConfig, data.CatalogId.ValueString(), data.Version.ValueString(), data.Architecture.ValueString())
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
	vms, createVmResponseDiag := apiclient.GetVms(apiCtx, hostConfig, "Name", data.Name.String())
	if createVmResponseDiag.HasError() {
		resp.Diagnostics.Append(createVmResponseDiag...)
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

		if specsDiag := common.CheckIfEnoughSpecs(apiCtx, hostConfig, data.Specs, architecture); specsDiag.HasError() {
			resp.Diagnostics.Append(specsDiag...)
			return
		}
	}

	if len(vms) > 0 {
		if !data.ForceChanges.ValueBool() {
			resp.Diagnostics.AddError("Vm already exists", "The vm "+data.Name.ValueString()+" already exists")
			return
		} else {
			// if we have force changes, we will remove the vm
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.Name.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
				return
			}
		}
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

	createVmResponse, createVmResponseDiag := apiclient.CreateVm(apiCtx, hostConfig, createMachineRequest)
	if createVmResponseDiag.HasError() {
		common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.Name.ValueString())
		resp.Diagnostics.Append(createVmResponseDiag...)
		return
	}

	data.ID = types.StringValue(createVmResponse.ID)
	tflog.Info(apiCtx, "Created vm with id "+data.ID.ValueString())

	createdVM, createVmResponseDiag := apiclient.GetVm(apiCtx, hostConfig, createVmResponse.ID)
	if createVmResponseDiag.HasError() {
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		resp.Diagnostics.Append(createVmResponseDiag...)
		return
	}

	if createdVM == nil {
		resp.Diagnostics.AddError("VM not found", "There was an issue creating the VM, we could not find it in the host")
		return
	}

	hostConfig.HostId = createdVM.HostId

	// stopping the machine as it might need some operations where the machine needs to be stopped
	// add anything here in sequence that needs to be done before the machine is started
	// so we do not loose time waiting for the machine to stop
	stoppedVm, createVmResponseDiag := common.EnsureMachineStopped(apiCtx, hostConfig, createdVM)
	if createVmResponseDiag.HasError() {
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		resp.Diagnostics.Append(createVmResponseDiag...)
	}

	// Applying the Specs block
	if data.Specs != nil {
		if diags := common.SpecsBlockOnCreate(apiCtx, hostConfig, stoppedVm, data.Specs); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}
	}

	// Configuring the machine if there is any configuration
	if vmBlockDiag := common.VmConfigBlockOnCreate(apiCtx, hostConfig, stoppedVm, data.Config); vmBlockDiag.HasError() {
		resp.Diagnostics.Append(vmBlockDiag...)
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		return
	}

	// Applying any prlctl commands
	if prlctlDiag := common.PrlCtlBlockOnCreate(apiCtx, hostConfig, stoppedVm, data.PrlCtl); prlctlDiag.HasError() {
		resp.Diagnostics.Append(prlctlDiag...)
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		return
	}

	// Processing shared folders
	if sharedFolderDiag := common.SharedFoldersBlockOnCreate(apiCtx, hostConfig, stoppedVm, data.SharedFolder); sharedFolderDiag.HasError() {
		resp.Diagnostics.Append(sharedFolderDiag...)
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		return
	}

	// Running the post processor scripts
	if postProcessDiag := common.RunPostProcessorScript(apiCtx, hostConfig, stoppedVm, data.PostProcessorScripts); postProcessDiag.HasError() {
		resp.Diagnostics.Append(postProcessDiag...)
		if data.ID.ValueString() != "" {
			if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
				resp.Diagnostics.Append(ensureRemoveDiag...)
			}
		}
		return
	}

	if len(data.ReverseProxyHosts) > 0 {
		rpHostConfig := hostConfig
		rpHostConfig.HostId = stoppedVm.HostId
		rpHosts, updateDiag := updateReverseProxyHostsTarget(apiCtx, &data, rpHostConfig, stoppedVm)
		if updateDiag.HasError() {
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			resp.Diagnostics.Append(updateDiag...)
			return
		}

		result, createDiag := reverseproxy.Create(apiCtx, rpHostConfig, rpHosts)
		if createDiag.HasError() {
			resp.Diagnostics.Append(createDiag...)

			if diag := reverseproxy.Delete(apiCtx, rpHostConfig, rpHosts); diag.HasError() {
				tflog.Error(apiCtx, "Error deleting reverse proxy hosts")
			}

			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}

		for i := range result {
			data.ReverseProxyHosts[i].ID = result[i].ID
		}
	}

	// Starting the vm by default, otherwise we will stop the VM from being created
	if data.RunAfterCreate.ValueBool() || data.KeepRunning.ValueBool() || (data.RunAfterCreate.IsUnknown() && data.KeepRunning.IsUnknown()) {
		if _, diag := common.EnsureMachineRunning(apiCtx, hostConfig, stoppedVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}

		_, diag := apiclient.GetVm(apiCtx, hostConfig, createVmResponse.ID)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}
	} else {
		// If we are not starting the machine, we will stop it
		if _, diag := common.EnsureMachineStopped(apiCtx, hostConfig, stoppedVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}
	}

	externalIp := ""
	internalIp := ""
	data.HostUrl = types.StringValue(stoppedVm.HostUrl)
	retryAttempts := 10
	var refreshVm *apimodels.VirtualMachine
	var refreshDiag diag.Diagnostics
	for {
		refreshVm, refreshDiag = apiclient.GetVm(apiCtx, hostConfig, stoppedVm.ID)
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
			mappedObject, diag := result.MapObject(apiCtx)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				if data.ID.ValueString() != "" {
					if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
						resp.Diagnostics.Append(ensureRemoveDiag...)
					}
				}
				return
			}

			elements = append(elements, mappedObject)
			listValue, diag := types.ListValue(result.ElementType(apiCtx), elements)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				if data.ID.ValueString() != "" {
					if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
						resp.Diagnostics.Append(ensureRemoveDiag...)
					}
				}
				return
			}

			script.Result = listValue
		}
	}

	// Setting the state of the vm to running if that is required
	if (data.RunAfterCreate.ValueBool() || data.RunAfterCreate.IsUnknown() || data.RunAfterCreate.IsNull()) && (refreshVm.State == "stopped") {
		if _, diag := common.EnsureMachineRunning(apiCtx, hostConfig, refreshVm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				if ensureRemoveDiag := common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString()); ensureRemoveDiag.HasError() {
					resp.Diagnostics.Append(ensureRemoveDiag...)
				}
			}
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(apiCtx, &data)...)
	if resp.Diagnostics.HasError() {
		if data.ID.ValueString() != "" {
			common.EnsureMachineIsRemoved(apiCtx, hostConfig, data.ID.ValueString())
		}
		return
	}
}

func (r *RemoteVmResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data models.RemoteVmResourceModelV2

	// Setting the default timeout
	contextTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aptCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(aptCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		aptCtx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(aptCtx, telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(aptCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

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

	vm, diag := apiclient.GetVm(aptCtx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(aptCtx)
		return
	}

	data.Name = types.StringValue(vm.Name)

	resp.Diagnostics.Append(req.State.Set(aptCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteVmResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data models.RemoteVmResourceModelV2
	var currentData models.RemoteVmResourceModelV2

	// Setting the default timeout
	contextTimeout, diags := data.Timeouts.Create(ctx, 60*time.Minute)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	aptCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(aptCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		aptCtx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(aptCtx, telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(aptCtx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(aptCtx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	vm, getVmDiag := apiclient.GetVm(aptCtx, hostConfig, currentData.ID.ValueString())
	if getVmDiag.HasError() {
		resp.Diagnostics.Append(getVmDiag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(aptCtx)
		return
	}
	currentVmState := vm.State

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

	configChanges := common.VmConfigBlockHasChanges(aptCtx, hostConfig, vm, data.Config, currentData.Config)
	specsChanges := common.SpecsBlockHasChanges(aptCtx, hostConfig, vm, data.Specs, currentData.Specs)
	prlctlChanges := common.PrlCtlBlockHasChanges(aptCtx, hostConfig, vm, data.PrlCtl, currentData.PrlCtl)
	postProcessorScriptChanges := common.PostProcessorHasChanges(aptCtx, data.PostProcessorScripts, currentData.PostProcessorScripts)
	if specsChanges || configChanges || prlctlChanges || nameChanges.HasChanges() {
		requireShutdown = true
	}

	if requireShutdown && vm.State != "stopped" {
		if data.ForceChanges.ValueBool() {
			if newVm, stopDiag := common.EnsureMachineStopped(aptCtx, hostConfig, vm); stopDiag.HasError() {
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
		_, nameChangeDiag := apiclient.ConfigureMachine(aptCtx, hostConfig, vm.ID, nameChanges)
		if nameChangeDiag.HasError() {
			resp.Diagnostics.Append(nameChangeDiag...)
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

		if specBlockDiag := common.SpecsBlockOnUpdate(aptCtx, hostConfig, vm, data.Specs, currentData.Specs); specBlockDiag.HasError() {
			resp.Diagnostics.Append(specBlockDiag...)
			return
		}
	}

	// Configuring the machine if there is any configuration
	if configChanges {
		if diag := common.VmConfigBlockOnUpdate(aptCtx, hostConfig, vm, data.Config, currentData.Config); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Applying any prlctl commands
	if prlctlChanges {
		if diag := common.PrlCtlBlockOnUpdate(aptCtx, hostConfig, vm, data.PrlCtl); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Restarting the machine if needed
	if needsRestart || (vm.State == "stopped" && currentState == "running") {
		if newVm, startDiags := common.EnsureMachineRunning(aptCtx, hostConfig, vm); startDiags.HasError() {
			resp.Diagnostics.Append(startDiags...)
			return
		} else {
			vm = newVm
		}
	}

	// Processing shared folders
	if diag := common.SharedFoldersBlockOnUpdate(aptCtx, hostConfig, vm, data.SharedFolder, currentData.SharedFolder); diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// Running the post processor scripts
	if postProcessorScriptChanges {
		if diag := common.RunPostProcessorScript(aptCtx, hostConfig, vm, data.PostProcessorScripts); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	} else {
		data.PostProcessorScripts = currentData.PostProcessorScripts
	}

	if reverseproxy.ReverseProxyHostsDiff(data.ReverseProxyHosts, currentData.ReverseProxyHosts) {
		copyCurrentRpHosts := reverseproxy.CopyReverseProxyHosts(currentData.ReverseProxyHosts)
		copyRpHosts := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)

		results, updateDiag := reverseproxy.Update(aptCtx, hostConfig, copyCurrentRpHosts, copyRpHosts)
		if updateDiag.HasError() {
			resp.Diagnostics.Append(updateDiag...)
			revertResults, _ := reverseproxy.Revert(aptCtx, hostConfig, copyCurrentRpHosts, copyRpHosts)
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
		if _, diag := common.EnsureMachineRunning(aptCtx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(aptCtx, hostConfig, vm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
			}
			return
		}

		_, diag := apiclient.GetVm(aptCtx, hostConfig, vm.ID)
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	} else {
		// If we are not starting the machine, we will stop it
		if _, diag := common.EnsureMachineStopped(aptCtx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			if data.ID.ValueString() != "" {
				// If we have an ID, we need to delete the machine
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
				if _, diag := common.EnsureMachineStopped(aptCtx, hostConfig, vm); diag.HasError() {
					resp.Diagnostics.Append(diag...)
					return
				}
				apiclient.DeleteVm(aptCtx, hostConfig, data.ID.ValueString())
			}
			return
		}
	}

	externalIp := ""
	internalIp := ""
	data.HostUrl = types.StringValue(vm.HostUrl)
	retryAttempts := 10
	var refreshVm *apimodels.VirtualMachine
	var refreshDiag diag.Diagnostics
	for {
		refreshVm, refreshDiag = apiclient.GetVm(aptCtx, hostConfig, vm.ID)
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
			mappedObject, diag := result.MapObject(aptCtx)
			if diag.HasError() {
				resp.Diagnostics.Append(diag...)
				return
			}

			elements = append(elements, mappedObject)
			listValue, diag := types.ListValue(result.ElementType(aptCtx), elements)
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
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
			if refreshVm.State == "paused" || refreshVm.State == "suspended" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
		case "stopped":
			if refreshVm.State == "running" || refreshVm.State == "paused" || refreshVm.State == "suspended" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStop)
			}
		case "paused":
			if refreshVm.State == "running" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
			if refreshVm.State == "stopped" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
		case "suspended":
			if refreshVm.State == "running" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpResume)
			}
			if refreshVm.State == "stopped" {
				apiclient.SetMachineState(aptCtx, hostConfig, data.ID.ValueString(), apiclient.MachineStateOpStart)
			}
		}
	}

	if (data.RunAfterCreate.ValueBool() || data.KeepRunning.ValueBool() || (data.RunAfterCreate.IsUnknown() && data.KeepRunning.IsUnknown())) &&
		(vm.State == "stopped") {
		if _, diag := common.EnsureMachineRunning(aptCtx, hostConfig, vm); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	tflog.Info(aptCtx, "Updated vm with id "+data.ID.ValueString()+" and name "+data.Name.ValueString())

	resp.Diagnostics.Append(resp.State.Set(aptCtx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteVmResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data models.RemoteVmResourceModelV2

	// Setting the default timeout
	contextTimeout, diags := data.Timeouts.Create(ctx, 90*time.Minute)

	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	apiCtx, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(apiCtx, &data)...)

	telemetrySvc := telemetry.Get(apiCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		apiCtx,
		r.provider.License.String(),
		telemetry.EventRemoteImage, telemetry.ModeDestroy,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(apiCtx, telemetryEvent)

	if resp.Diagnostics.HasError() {
		return
	}

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

	vm, diag := apiclient.GetVm(apiCtx, hostConfig, data.ID.ValueString())
	if diag.HasError() {
		resp.Diagnostics.Append(diag...)
		return
	}

	// Nothing to do, machine does not exist
	if vm == nil {
		resp.Diagnostics.Append(req.State.Set(apiCtx, &data)...)
		return
	} else {
		hostConfig.HostId = vm.HostId
	}

	// Running cleanup script if any
	if data.OnDestroyScript != nil {
		_ = common.RunPostProcessorScript(apiCtx, hostConfig, vm, data.OnDestroyScript)
	}

	if len(data.ReverseProxyHosts) > 0 {
		rpHostConfig := hostConfig
		rpHostConfig.HostId = vm.HostId
		rpHostsCopy := reverseproxy.CopyReverseProxyHosts(data.ReverseProxyHosts)
		if diag := reverseproxy.Delete(apiCtx, rpHostConfig, rpHostsCopy); diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
	}

	// Stopping the machine
	if _, stopDiag := common.EnsureMachineStopped(apiCtx, hostConfig, vm); stopDiag.HasError() {
		resp.Diagnostics.Append(stopDiag...)
		return
	}

	deleteDiag := apiclient.DeleteVm(apiCtx, hostConfig, vm.ID)
	if deleteDiag.HasError() {
		resp.Diagnostics.Append(deleteDiag...)
		return
	}

	retryCount := 0
	for {
		vm, diag := apiclient.GetVm(apiCtx, hostConfig, data.ID.ValueString())
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

	resp.Diagnostics.Append(req.State.Set(apiCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *RemoteVmResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RemoteVmResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	v0Schema := schemas.GetRemoteImageSchemaV0(ctx)
	V1Schema := schemas.GetRemoteImageSchemaV1(ctx)
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &v0Schema,
			StateUpgrader: UpgradeStateToV1,
		},
		1: {
			PriorSchema:   &V1Schema,
			StateUpgrader: UpgradeStateToV2,
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgradedStateData)...)
}

func UpgradeStateToV2(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var priorStateData models.RemoteVmResourceModelV1
	resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	upgradedStateData := models.RemoteVmResourceModelV2{
		Authenticator:        priorStateData.Authenticator,
		Host:                 priorStateData.Host,
		HostUrl:              types.StringUnknown(),
		Orchestrator:         priorStateData.Orchestrator,
		ID:                   priorStateData.ID,
		OsType:               priorStateData.OsType,
		ExternalIp:           priorStateData.ExternalIp,
		InternalIp:           priorStateData.InternalIp,
		OrchestratorHostId:   priorStateData.OrchestratorHostId,
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
		KeepRunning:          priorStateData.KeepRunning,
		ReverseProxyHosts:    priorStateData.ReverseProxyHosts,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgradedStateData)...)
}

func updateReverseProxyHostsTarget(ctx context.Context, data *models.RemoteVmResourceModelV2, hostConfig apiclient.HostConfig, targetVm *apimodels.VirtualMachine) ([]reverseproxy.ReverseProxyHost, diag.Diagnostics) {
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
