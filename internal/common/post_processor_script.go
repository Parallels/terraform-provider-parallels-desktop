package common

import (
	"context"
	"fmt"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func RunPostProcessorScript(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, scripts []*postprocessorscript.PostProcessorScript) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	refreshVm, diag := apiclient.GetVm(ctx, hostConfig, vm.ID)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	tflog.Info(ctx, "Running post processor script on vm "+refreshVm.Name+" with state "+refreshVm.State+"...")
	if refreshVm.State != "running" {
		if result, diag := apiclient.SetMachineState(ctx, hostConfig, refreshVm.ID, apiclient.MachineStateOpStart); diag.HasError() {
			tflog.Error(ctx, fmt.Sprintf("Error starting vm: %v", result))
			diagnostics.Append(diag...)
			return diagnostics
		}

		tflog.Info(ctx, "Waiting for vm to start...")
		time.Sleep(time.Minute * 1)
	}

	for _, script := range scripts {
		resultDiag := script.Apply(ctx, hostConfig, refreshVm.ID)
		tflog.Info(ctx, fmt.Sprintf("Script %s executed, result %v", script.Inline, resultDiag))
		if resultDiag.HasError() {
			tflog.Info(ctx, "Error running post processor script")
			diagnostics.Append(resultDiag...)
			return diagnostics
		}
	}

	return diagnostics
}
