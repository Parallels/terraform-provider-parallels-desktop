package common

import (
	"context"
	"strconv"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func CheckIfEnoughSpecs(ctx context.Context, hostConfig apiclient.HostConfig, specs *vmspecs.VmSpecs) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// checking if we have enough resources for this change
	hardwareInfo, diag := apiclient.GetSystemUsage(ctx, hostConfig)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	var updateCpuValueInt int
	var updateMemoryValueInt int
	var err error
	if specs.CpuCount.ValueString() == "" {
		updateCpuValueInt = 2
	} else {
		updateCpuValueInt, err = strconv.Atoi(specs.CpuCount.ValueString())
	}
	if err != nil {
		diagnostics.AddError("error converting cpu count", err.Error())
		return diagnostics
	}
	if hardwareInfo.TotalAvailable.LogicalCpuCount-int64(updateCpuValueInt) <= 0 {
		diagnostics.AddError("not enough cpus", "not enough cpus")
		return diagnostics
	}

	if specs.MemorySize.ValueString() == "" {
		updateMemoryValueInt = 2048
	} else {
		updateMemoryValueInt, err = strconv.Atoi(specs.MemorySize.ValueString())
	}
	if err != nil {
		diagnostics.AddError("error converting memory size", err.Error())
		return diagnostics
	}
	if hardwareInfo.TotalAvailable.MemorySize-float64(updateMemoryValueInt) <= 0 {
		diagnostics.AddError("not enough memory", "not enough memory")
		return diagnostics
	}

	return diagnostics
}
