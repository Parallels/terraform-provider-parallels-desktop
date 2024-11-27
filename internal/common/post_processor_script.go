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

	currentVm, diag := EnsureMachineRunning(ctx, hostConfig, vm)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return diagnostics
	}
	if currentVm == nil {
		diagnostics.AddError("There was an error getting the vm", "vm is nil")
		return diagnostics
	}

	tflog.Info(ctx, "Running post processor script on vm "+currentVm.Name+" with state "+currentVm.State+"...")

	for _, script := range scripts {
		maxRetries := constants.DEFAULT_SCRIPT_MAX_RETRY_COUNT
		waitBeforeRetry := time.Second * time.Duration(constants.DEFAULT_SCRIPT_RETRY_INTERVAL_IN_SECONDS)

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
			tflog.Info(ctx, fmt.Sprintf("Running post processor script %s on vm %s with state %s [%v]", script.Inline, currentVm.Name, currentVm.State, maxRetries))
			resultDiag := script.Apply(ctx, hostConfig, vm)
			tflog.Info(ctx, fmt.Sprintf("Script %s executed, result %v", script.Inline, resultDiag))
			if resultDiag.HasError() {
				errorMessages := "Script failed to run:"
				for _, diag := range resultDiag.Errors() {
					errorMessages += "\n" + diag.Summary() + " " + diag.Detail()
				}
				return errors.New(errorMessages)
			}

			return nil
		}); err != nil {
			tflog.Info(ctx, fmt.Sprintf("Error running post processor script %s on vm %s with state %s [%v]", script.Inline, currentVm.Name, currentVm.State, maxRetries))
			tflog.Info(ctx, "Error running post processor script")
			diagnostics.AddError("Error running post processor script", err.Error())
			return diagnostics
		}
	}

	return diagnostics
}

func PostProcessorHasChanges(ctx context.Context, planPostProcessorScript, statePostProcessorScript []*postprocessorscript.PostProcessorScript) bool {
	for i, script := range planPostProcessorScript {
		if script.AlwaysRunOnUpdate.ValueBool() {
			return true
		}
		innerElements := script.Inline.Elements()
		if len(innerElements) > 0 && len(statePostProcessorScript) == 0 {
			return true
		}
		if len(innerElements) != len(statePostProcessorScript[i].Inline.Elements()) {
			return true
		}
		for j, element := range innerElements {
			g := element.String()
			if g != statePostProcessorScript[i].Inline.Elements()[j].String() {
				return true
			}
		}
	}

	return false
}
