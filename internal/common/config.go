package common

import (
	"context"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/vmconfig"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func VmConfigBlockOnCreate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, config *vmconfig.VmConfig) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	if config != nil {
		if config.StartHeadless.ValueBool() {
			if diag := config.ApplyStartHeadless(ctx, hostConfig, *vm); diag.HasError() {
				diagnostics.Append(diag...)
			}
		}
		if config.EnableRosetta.ValueBool() {
			if diag := config.ApplyEnableRosetta(ctx, hostConfig, *vm); diag.HasError() {
				diagnostics.Append(diag...)
			}
		}
		if config.PauseIdle.ValueBool() {
			if diag := config.ApplyPauseIdle(ctx, hostConfig, *vm); diag.HasError() {
				diagnostics.Append(diag...)
			}
		}
		if config.AutoStartOnHost.ValueBool() {
			if diag := config.ApplyAutoStartOnHost(ctx, hostConfig, *vm); diag.HasError() {
				diagnostics.Append(diag...)
			}
		}
	}
	return diagnostics
}

func VmConfigBlockOnUpdate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planConfig, stateConfig *vmconfig.VmConfig) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	if planConfig != nil && stateConfig == nil {
		tflog.Info(ctx, "Current vm config needs updating as there is a new config that does not exist in the current state")
		diag := VmConfigBlockOnCreate(ctx, hostConfig, vm, planConfig)
		return diag
	}

	if planConfig == nil && stateConfig != nil {
		tflog.Info(ctx, "Current vm config needs updating as the new config is null")
		config := &vmconfig.VmConfig{
			StartHeadless:   types.BoolValue(false),
			EnableRosetta:   types.BoolValue(false),
			PauseIdle:       types.BoolValue(false),
			AutoStartOnHost: types.BoolValue(false),
		}
		if diag := config.ApplyStartHeadless(ctx, hostConfig, *vm); diag.HasError() {
			diagnostics.Append(diag...)
		}
		if diag := config.ApplyEnableRosetta(ctx, hostConfig, *vm); diag.HasError() {
			diagnostics.Append(diag...)
		}
		if diag := config.ApplyPauseIdle(ctx, hostConfig, *vm); diag.HasError() {
			diagnostics.Append(diag...)
		}
		if diag := config.ApplyAutoStartOnHost(ctx, hostConfig, *vm); diag.HasError() {
			diagnostics.Append(diag...)
		}

		return diagnostics
	}

	if planConfig != nil && stateConfig != nil {
		if planConfig.StartHeadless.ValueBool() != stateConfig.StartHeadless.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the start_headless value has changed")
			diag := planConfig.ApplyStartHeadless(ctx, hostConfig, *vm)
			diagnostics.Append(diag...)
		}
		if planConfig.EnableRosetta.ValueBool() != stateConfig.EnableRosetta.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the enable_rosetta value has changed")
			diag := planConfig.ApplyEnableRosetta(ctx, hostConfig, *vm)
			diagnostics.Append(diag...)
		}
		if planConfig.PauseIdle.ValueBool() != stateConfig.PauseIdle.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the pause_idle value has changed")
			diag := planConfig.ApplyPauseIdle(ctx, hostConfig, *vm)
			diagnostics.Append(diag...)
		}
		if planConfig.AutoStartOnHost.ValueBool() != stateConfig.AutoStartOnHost.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the auto_start_on_host value has changed")
			diag := planConfig.ApplyAutoStartOnHost(ctx, hostConfig, *vm)
			diagnostics.Append(diag...)
		}
	}

	return diagnostics
}

func VmConfigBlockHasChanges(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planConfig, stateConfig *vmconfig.VmConfig) bool {
	if planConfig == nil && stateConfig == nil {
		return false
	}

	if planConfig != nil && stateConfig == nil {
		tflog.Info(ctx, "Current vm config needs updating as there is a new config that does not exist in the current state")
		return true
	}

	if planConfig == nil && stateConfig != nil {
		tflog.Info(ctx, "Current vm config needs updating as the new config is null")
		return true
	}

	if planConfig != nil && stateConfig != nil {
		if planConfig.StartHeadless.ValueBool() != stateConfig.StartHeadless.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the start_headless value has changed")
			return true
		}
		if planConfig.EnableRosetta.ValueBool() != stateConfig.EnableRosetta.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the enable_rosetta value has changed")
			return true
		}
		if planConfig.PauseIdle.ValueBool() != stateConfig.PauseIdle.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the pause_idle value has changed")
			return true
		}
		if planConfig.AutoStartOnHost.ValueBool() != stateConfig.AutoStartOnHost.ValueBool() {
			tflog.Info(ctx, "Current vm config needs updating as the auto_start_on_host value has changed")
			return true
		}
	}

	return false
}
