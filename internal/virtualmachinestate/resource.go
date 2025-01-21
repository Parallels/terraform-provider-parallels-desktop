package virtualmachinestate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/telemetry"
	resource_models "terraform-provider-parallels-desktop/internal/virtualmachinestate/models"
	"terraform-provider-parallels-desktop/internal/virtualmachinestate/schemas"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_                  resource.Resource                = &VirtualMachineStateResource{}
	_                  resource.ResourceWithImportState = &VirtualMachineStateResource{}
	maxRetries                                          = 10
	waitBetweenRetries                                  = 10 * time.Second
)

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
	resp.Schema = schemas.VirtualMachineStateResourceSchemaV1
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
	var data resource_models.VirtualMachineStateResourceModelV1

	// Setting the default timeout
	ctxTimeout := 10 * time.Minute

	apiCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(apiCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		apiCtx,
		r.provider.License.String(),
		telemetry.EventVirtualMachineState, telemetry.ModeCreate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(apiCtx, telemetryEvent)

	resp.Diagnostics.Append(req.Plan.Get(apiCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !r.IsValidOperation(data.Operation.ValueString()) {
		resp.Diagnostics.AddError("Invalid operation", fmt.Sprintf("Operation %s is not valid", data.Operation.ValueString()))
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
	if vm == nil {
		resp.State.RemoveResource(apiCtx)
		return
	}

	isInState, err := r.IsInDesiredState(vm, data.Operation.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error changing the machine state", err.Error())
		return
	}

	if !isInState {
		result, diag := apiclient.SetMachineState(apiCtx, hostConfig, data.ID.ValueString(), r.GetOpState(data.Operation.ValueString()))
		if diag.HasError() {
			resp.Diagnostics.Append(diag...)
			return
		}
		if !result {
			resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
			return
		}

		if data.EnsureState.ValueBool() {
			if r.GetDesiredState(data.Operation.ValueString()) == "" {
				resp.Diagnostics.AddError("error changing the machine state", "Could not determine the desired state")
				return
			}

			retries := 0
			for {
				refreshedVm, refreshDiag := apiclient.GetVm(apiCtx, hostConfig, data.ID.ValueString())
				if diag.HasError() {
					resp.Diagnostics.Append(refreshDiag...)
					return
				}
				if refreshedVm.State == r.GetDesiredState(data.Operation.ValueString()) {
					vm = refreshedVm
					break
				}
				if retries >= maxRetries {
					resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
					return
				}
				retries++
				time.Sleep(waitBetweenRetries)
			}

			data.CurrentState = types.StringValue(vm.State)
		} else {
			data.CurrentState = data.Operation
		}

		data.Operation = types.StringValue(data.Operation.ValueString())
		data.CurrentState = types.StringValue(vm.State)
	} else {
		data.Operation = types.StringValue(data.Operation.ValueString())
		data.CurrentState = types.StringValue(vm.State)
	}

	tflog.Trace(apiCtx, "virtual machine "+vm.Name+" state changed to "+data.Operation.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(apiCtx, &data)...)
}

func (r *VirtualMachineStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_models.VirtualMachineStateResourceModelV1

	// Setting the default timeout
	ctxTimeout := 10 * time.Minute

	apiCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(apiCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		apiCtx,
		r.provider.License.String(),
		telemetry.EventVirtualMachineState, telemetry.ModeRead,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(apiCtx, telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(apiCtx, &data)...)

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
	if vm == nil {
		resp.State.RemoveResource(apiCtx)
		return
	}

	data.CurrentState = types.StringValue(vm.State)

	resp.Diagnostics.Append(req.State.Get(apiCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data resource_models.VirtualMachineStateResourceModelV1
	var currentData resource_models.VirtualMachineStateResourceModelV1

	// Setting the default timeout
	ctxTimeout := 10 * time.Minute

	ctx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(ctx)
	telemetryEvent := telemetry.NewTelemetryItem(
		ctx,
		r.provider.License.String(),
		telemetry.EventVirtualMachineState, telemetry.ModeUpdate,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(ctx, telemetryEvent)

	resp.Diagnostics.Append(req.State.Get(ctx, &currentData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !r.IsValidOperation(data.Operation.ValueString()) {
		resp.Diagnostics.AddError("Invalid operation", fmt.Sprintf("Operation %s is not valid", data.Operation.ValueString()))
		return
	}

	if data.ID.IsNull() {
		resp.Diagnostics.AddError("Id is required", "Id is required")
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

	vm, vmDiag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
	if vmDiag.HasError() {
		resp.Diagnostics.Append(vmDiag...)
		return
	}
	if vm == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	isInState, err := r.IsInDesiredState(vm, data.Operation.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error changing the machine state", err.Error())
		return
	}

	if !isInState && currentData.Operation.ValueString() != data.Operation.ValueString() {
		result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, data.ID.ValueString(), r.GetOpState(data.Operation.ValueString()))
		if stateDiag.HasError() {
			resp.Diagnostics.Append(stateDiag...)
			return
		}
		if !result {
			resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
			return
		}

		if data.EnsureState.ValueBool() {
			if r.GetDesiredState(data.Operation.ValueString()) == "" {
				resp.Diagnostics.AddError("error changing the machine state", "Could not determine the desired state")
				return
			}

			retries := 0
			for {
				refreshedVm, refreshDiag := apiclient.GetVm(ctx, hostConfig, data.ID.ValueString())
				if refreshDiag.HasError() {
					resp.Diagnostics.Append(refreshDiag...)
					return
				}
				if refreshedVm.State == r.GetDesiredState(data.Operation.ValueString()) {
					if refreshedVm.State == "running" {
						echoHelloCommand := apimodels.PostScriptItem{
							Command:          "echo 'I am running'",
							VirtualMachineId: refreshedVm.ID,
						}

						// Only breaking out of the loop if the script executes successfully
						if _, execDiag := apiclient.ExecuteScript(ctx, hostConfig, echoHelloCommand); !execDiag.HasError() {
							tflog.Info(ctx, "Machine "+vm.Name+" is running")
							resp.Diagnostics = diag.Diagnostics{}
							vm = refreshedVm
							break
						}
					} else {
						vm = refreshedVm
						break
					}
				}
				if retries >= maxRetries {
					resp.Diagnostics.AddError("error changing the machine state", "Could not change the machine "+vm.Name+" state to +"+data.Operation.ValueString())
					return
				}
				retries++
				time.Sleep(waitBetweenRetries)
			}

			data.CurrentState = types.StringValue(vm.State)
		} else {
			data.CurrentState = data.Operation
		}
	} else {
		data.CurrentState = types.StringValue(vm.State)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_models.VirtualMachineStateResourceModelV1

	// Setting the default timeout
	ctxTimeout := 10 * time.Minute

	apiCtx, cancel := context.WithTimeout(ctx, ctxTimeout)
	defer cancel()

	telemetrySvc := telemetry.Get(apiCtx)
	telemetryEvent := telemetry.NewTelemetryItem(
		apiCtx,
		r.provider.License.String(),
		telemetry.EventVirtualMachineState, telemetry.ModeDestroy,
		nil,
		nil,
	)
	telemetrySvc.TrackEvent(apiCtx, telemetryEvent)

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(apiCtx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *VirtualMachineStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VirtualMachineStateResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	v0Schema := schemas.VirtualMachineStateResourceSchemaV0
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &v0Schema,
			StateUpgrader: UpgradeStateToV1,
		},
	}
}

func UpgradeStateToV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var priorStateData resource_models.VirtualMachineStateResourceModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	upgradedStateData := resource_models.VirtualMachineStateResourceModelV1{
		Authenticator: priorStateData.Authenticator,
		Host:          priorStateData.Host,
		Orchestrator:  types.StringUnknown(),
		ID:            priorStateData.ID,
		Operation:     priorStateData.Operation,
		CurrentState:  priorStateData.CurrentState,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &upgradedStateData)...)
}

func (r *VirtualMachineStateResource) IsInDesiredState(vm *apimodels.VirtualMachine, desiredState string) (bool, error) {
	switch strings.ToLower(desiredState) {
	case "start":
		if vm.State == "suspended" ||
			vm.State == "suspending" ||
			vm.State == "paused" ||
			vm.State == "pausing" {
			return false, errors.New("cannot start a machine that is suspended, suspending, paused, or pausing")
		}
		return vm.State == "running" || vm.State == "starting", nil
	case "stop":
		return vm.State == "stopped" || vm.State == "stopping", nil
	case "suspend":
		if vm.State == "paused" ||
			vm.State == "pausing" ||
			vm.State == "stopped" ||
			vm.State == "stopping" {
			return false, errors.New("cannot suspend a machine that is paused, pausing, stopped, or stopping")
		}
		return vm.State == "suspended" || vm.State == "suspending", nil
	case "pause":
		if vm.State == "suspended" ||
			vm.State == "suspending" ||
			vm.State == "stopped" ||
			vm.State == "stopping" {
			return false, errors.New("cannot pause a machine that is suspended, suspending, stopped, or stopping")
		}
		return vm.State == "paused" || vm.State == "pausing", nil
	case "resume":
		if vm.State == "stopped" ||
			vm.State == "stopping" {
			return false, errors.New("cannot resume a machine that is stopped or stopping")
		}
		return vm.State == "running" || vm.State == "paused" || vm.State == "suspended", nil
	default:
		return false, nil
	}
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

func (r *VirtualMachineStateResource) IsValidOperation(value string) bool {
	switch strings.ToLower(value) {
	case "start", "stop", "suspend", "pause", "resume", "restart":
		return true
	default:
		return false
	}
}

func (r *VirtualMachineStateResource) GetDesiredState(operation string) string {
	var desiredState string
	switch strings.ToLower(operation) {
	case "start":
		desiredState = "running"
	case "stop":
		desiredState = "stopped"
	case "suspend":
		desiredState = "suspended"
	case "pause":
		desiredState = "paused"
	case "resume":
		desiredState = "running"
	case "restart":
		desiredState = "running"
	default:
		desiredState = ""
	}

	return desiredState
}
