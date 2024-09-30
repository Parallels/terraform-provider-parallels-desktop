package vmconfig

import (
	"context"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type VmConfig struct {
	StartHeadless   types.Bool `tfsdk:"start_headless"`
	EnableRosetta   types.Bool `tfsdk:"enable_rosetta"`
	PauseIdle       types.Bool `tfsdk:"pause_idle"`
	AutoStartOnHost types.Bool `tfsdk:"auto_start_on_host"`
}

func (s *VmConfig) Schema() map[string]schema.Attribute {
	return SchemaBlock.Attributes
}

func (s *VmConfig) Elements() []attr.Value {
	attrs := []attr.Value{
		s.StartHeadless,
		s.EnableRosetta,
		s.PauseIdle,
		s.AutoStartOnHost,
	}

	return attrs
}

func (s *VmConfig) ElementType() attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"start_headless":     types.BoolType,
			"enable_rosetta":     types.BoolType,
			"pause_idle":         types.BoolType,
			"auto_start_on_host": types.BoolType,
		},
	}
}

func (s *VmConfig) MapObject() (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["start_headless"] = types.BoolType
	attributeTypes["enable_rosetta"] = types.BoolType
	attributeTypes["pause_idle"] = types.BoolType
	attributeTypes["auto_start_on_host"] = types.BoolType

	attrs := map[string]attr.Value{}
	attrs["start_headless"] = s.StartHeadless
	attrs["enable_rosetta"] = s.EnableRosetta
	attrs["pause_idle"] = s.PauseIdle
	attrs["auto_start_on_host"] = s.AutoStartOnHost

	return types.ObjectValue(attributeTypes, attrs)
}

func (s *VmConfig) ApplyStartHeadless(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	return s.toggle(ctx, config, vm, "start_headless")
}

func (s *VmConfig) ApplyEnableRosetta(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	return s.toggle(ctx, config, vm, "enable_rosetta")
}

func (s *VmConfig) ApplyPauseIdle(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	return s.toggle(ctx, config, vm, "pause_idle")
}

func (s *VmConfig) ApplyAutoStartOnHost(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine) diag.Diagnostics {
	return s.toggle(ctx, config, vm, "auto_start_on_host")
}

func (s *VmConfig) toggle(ctx context.Context, config apiclient.HostConfig, vm apimodels.VirtualMachine, feature string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	diagnostics := diag.Diagnostics{}
	refreshVm, diag := apiclient.GetVm(ctx, config, vm.ID)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	if refreshVm.State != "stopped" {
		result, stateDiag := apiclient.SetMachineState(ctx, config, refreshVm.ID, apiclient.MachineStateOpStop)
		if stateDiag.HasError() {
			diagnostic.Append(stateDiag...)
		}
		if !result {
			diagnostic.AddError("error stopping vm", "error stopping vm")
		}
		tflog.Info(ctx, "Waiting for vm "+refreshVm.Name+" to stop")
	}

	vmConfigRequest := apimodels.NewVmConfigRequest(refreshVm.User)
	switch feature {
	case "start_headless":
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("cmd")
		op.WithOperation("set")
		if s.StartHeadless.ValueBool() {
			op.WithOption("startup-view", "headless")
		} else {
			op.WithOption("startup-view", "window")
		}
		op.Append()
	case "enable_rosetta":
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("cmd")
		op.WithOperation("set")
		if s.StartHeadless.ValueBool() {
			op.WithOption("rosetta-linux", "on")
		} else {
			op.WithOption("rosetta-linux", "off")
		}
		op.Append()
	case "pause_idle":
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("cmd")
		op.WithOperation("set")
		if s.StartHeadless.ValueBool() {
			op.WithOption("pause-idle", "on")
		} else {
			op.WithOption("pause-idle", "off")
		}
		op.Append()
	case "auto_start_on_host":
		op := apimodels.NewVmConfigRequestOperation(vmConfigRequest)
		op.WithGroup("cmd")
		op.WithOperation("set")
		if s.StartHeadless.ValueBool() {
			op.WithOption("autostart", "start-host")
		} else {
			op.WithOption("autostart", "off")
		}
		op.Append()
	}

	_, resultDiagnostic := apiclient.ConfigureMachine(ctx, config, refreshVm.ID, vmConfigRequest)
	if resultDiagnostic.HasError() {
		diagnostic.Append(resultDiagnostic...)
		return diagnostic
	}

	return diagnostic
}
