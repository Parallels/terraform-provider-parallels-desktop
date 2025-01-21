package common

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func CheckIfEnoughSpecs(ctx context.Context, hostConfig apiclient.HostConfig, specs *vmspecs.VmSpecs, arch string) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	// checking if we have enough resources for this change
	var hardwareInfo *apimodels.SystemUsageResponse
	var diag diag.Diagnostics

	if hostConfig.IsOrchestrator {
		if arch == "" {
			arch = "arm64"
		}

		// GetOrchestratorResources is a function that returns the orchestrator resources
		orchestratorResources, orchestratorDiag := apiclient.GetOrchestratorResources(ctx, hostConfig)
		if orchestratorDiag.HasError() {
			diagnostics.Append(orchestratorDiag...)
			return diagnostics
		}

		foundArchitectureResources := false
		for _, orchestratorResource := range orchestratorResources {
			if strings.EqualFold(orchestratorResource.CpuType, arch) {
				hardwareInfo = orchestratorResource
				foundArchitectureResources = true
				break
			}
		}
		if !foundArchitectureResources {
			diagnostics.AddError("Hardware", fmt.Sprintf("Did not find any hosts for %s architecture in the orchestrator, please check if you have any online", arch))
			return diagnostics
		}
	} else {
		hardwareInfo, diag = apiclient.GetSystemUsage(ctx, hostConfig)
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
	}

	if hardwareInfo == nil {
		if diagnostics.HasError() {
			return diagnostics
		} else {
			diagnostics.AddError("error getting hardware info", "error getting hardware info, hardware info is nil")
		}
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
	if hardwareInfo.TotalAvailable.LogicalCpuCount-int64(updateCpuValueInt) < 0 {
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
	if hardwareInfo.TotalAvailable.MemorySize-float64(updateMemoryValueInt) < 0 {
		diagnostics.AddError("not enough memory", "not enough memory")
		return diagnostics
	}

	return diagnostics
}
