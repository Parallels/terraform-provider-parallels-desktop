package common

import (
	"context"
	"fmt"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func RunPostProcessorScript(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, scripts []*postprocessorscript.PostProcessorScript) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	refreshVm, diag := EnsureMachineRunning(ctx, hostConfig, vm)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	tflog.Info(ctx, "Running post processor script on vm "+refreshVm.Name+" with state "+refreshVm.State+"...")

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
