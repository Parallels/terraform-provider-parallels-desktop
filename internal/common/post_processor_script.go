package common

import (
	"context"
	"errors"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/constants"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/retry"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func RunPostProcessorScript(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, scripts []*postprocessorscript.PostProcessorScript) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	if len(scripts) == 0 {
		return diagnostics
	}

	refreshVm, diag := EnsureMachineRunning(ctx, hostConfig, vm)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}

	tflog.Info(ctx, "Running post processor script on vm "+refreshVm.Name+" with state "+refreshVm.State+"...")

	for _, script := range scripts {
		maxRetries := constants.DEFAULT_MAX_RETRY_COUNT
		waitBeforeRetry := time.Second * time.Duration(constants.DEFAULT_RETRY_INTERVAL_IN_SECONDS)
		if script.Retry != nil {
			if !diag.HasError() {
				retriesMaxRetries := int(script.Retry.Attempts.ValueInt64())
				if retriesMaxRetries > 0 {
					maxRetries = retriesMaxRetries
				}

				retriesWaitBeforeRetry, err := helpers.ParseDuration(script.Retry.WaitBetweenAttempts.ValueString())
				if err == nil {
					waitBeforeRetry = retriesWaitBeforeRetry
				}
			}
		}

		if err := retry.For(maxRetries, waitBeforeRetry, func() error {
			tflog.Info(ctx, fmt.Sprintf("Running post processor script %s on vm %s with state %s [%v]", script.Inline, refreshVm.Name, refreshVm.State, maxRetries))
			resultDiag := script.Apply(ctx, hostConfig, refreshVm.ID)
			tflog.Info(ctx, fmt.Sprintf("Script %s executed, result %v", script.Inline, resultDiag))
			if resultDiag.HasError() {
				return errors.New("script failed")
			}

			return nil
		}); err != nil {
			tflog.Info(ctx, fmt.Sprintf("Error running post processor script %s on vm %s with state %s [%v]", script.Inline, refreshVm.Name, refreshVm.State, maxRetries))
			tflog.Info(ctx, "Error running post processor script")
			diagnostics.AddError("Error running post processor script", err.Error())
			return diagnostics
		}
	}

	return diagnostics
}
