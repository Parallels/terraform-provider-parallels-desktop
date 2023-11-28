package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func SpecsBlockOnCreate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, specs *vmspecs.VmSpecs) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	refreshVm, diag := EnsureMachineStopped(ctx, hostConfig, vm)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	specsDiagnostics := specs.Apply(ctx, hostConfig, *refreshVm)
	if specsDiagnostics.HasError() {
		diagnostics.Append(specsDiagnostics...)
	}
	return diagnostics
}

func SpecsBlockOnUpdate(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planSpecs, stateSpecs *vmspecs.VmSpecs) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// checking if we have enough resources for this change
	hardwareInfo, hardwareDiags := apiclient.GetSystemUsage(ctx, hostConfig)
	if hardwareDiags.HasError() {
		diagnostics.Append(hardwareDiags...)
		return diagnostics
	}

	changes := apimodels.NewVmConfigRequest(vm.User)
	if vm.State == "running" {
		// Because this is an update we need to take into account the already existing cpu and add it to the total available
		// if the vm is running, otherwise we already added that value to the reserved resources
		hardwareInfo.TotalAvailable.LogicalCpuCount = hardwareInfo.TotalAvailable.LogicalCpuCount + vm.Hardware.CPU.Cpus
		currentMemoryUsage, err := helpers.GetSizeByteFromString(vm.Hardware.Memory.Size)
		if err != nil {
			diagnostics.AddError("error getting memory size", err.Error())
			return diagnostics
		}
		hardwareInfo.TotalAvailable.MemorySize = hardwareInfo.TotalAvailable.MemorySize + helpers.ConvertByteToMegabyte(currentMemoryUsage)
	}

	if planSpecs.CpuCount.ValueString() != fmt.Sprintf("%v", vm.Hardware.CPU.Cpus) {
		updateValue := planSpecs.CpuCount.ValueString()
		if updateValue == "" {
			updateValue = "2"
		}

		updateValueInt, err := strconv.Atoi(updateValue)
		if err != nil {
			diagnostics.AddError("error converting cpu count", err.Error())
			return diagnostics
		}

		if hardwareInfo.TotalAvailable.LogicalCpuCount-int64(updateValueInt) <= 0 {
			diagnostics.AddError("not enough cpus", "not enough cpus")
			return diagnostics
		}

		op := apimodels.NewVmConfigRequestOperation(changes)
		op.WithGroup("cpu")
		op.WithOperation("set")
		op.WithValue(updateValue)
		if stateSpecs == nil || stateSpecs.CpuCount.ValueString() != updateValue {
			op.Append()
		}
	}

	if planSpecs.MemorySize.ValueString() != "" && planSpecs.MemorySize.ValueString() != strings.ReplaceAll(vm.Hardware.Memory.Size, "Mb", "") {
		updateValue := planSpecs.CpuCount.ValueString()
		if updateValue == "" {
			updateValue = "2048"
		}

		updateValueInt, err := strconv.Atoi(updateValue)
		if err != nil {
			diagnostics.AddError("error converting memory size", err.Error())
			return diagnostics
		}

		if hardwareInfo.TotalAvailable.MemorySize-float64(updateValueInt) <= 0 {
			diagnostics.AddError("not enough memory", "not enough memory")
			return diagnostics
		}

		op := apimodels.NewVmConfigRequestOperation(changes)
		op.WithGroup("memory")
		op.WithOperation("set")
		op.WithValue(updateValue)
		if stateSpecs == nil || stateSpecs.MemorySize.ValueString() != updateValue {
			op.Append()
		}
	}

	if changes.HasChanges() {
		if _, changesDiags := apiclient.ConfigureMachine(ctx, hostConfig, vm.ID, changes); changesDiags.HasError() {
			diagnostics.Append(changesDiags...)
			return diagnostics
		}
	}
	return diagnostics
}

func SpecsBlockHasChanges(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, planSpecs, stateSpecs *vmspecs.VmSpecs) bool {
	if planSpecs == nil && stateSpecs == nil {
		return false
	}

	if planSpecs == nil && stateSpecs != nil {
		return true
	}

	if planSpecs != nil && stateSpecs == nil {
		return true
	}

	if planSpecs != nil && stateSpecs != nil {
		if planSpecs.CpuCount.ValueString() != stateSpecs.CpuCount.ValueString() {
			return true
		}
		if planSpecs.MemorySize.ValueString() != stateSpecs.MemorySize.ValueString() {
			return true
		}
	}

	return false
}
