package common

import (
	"context"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/constants"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func EnsureMachineIsRemoved(ctx context.Context, hostConfig apiclient.HostConfig, vmIdOrName string) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	maxRetries := constants.DEFAULT_OPERATION_MAX_RETRY_COUNT
	vmIdOrName = strings.ReplaceAll(vmIdOrName, "\"", "")
	retryCount := 0
	for {
		diagnostics = diag.Diagnostics{}
		retryCount += 1

		// We have run out of retries, add an error and break out of the loop
		if retryCount >= maxRetries {
			diagnostics.AddError("Error starting vm", "Could not verify the state of the machine after starting, retry count exceeded")
			break
		}

		currentVm, refreshDiags := apiclient.GetVm(ctx, hostConfig, vmIdOrName)
		if refreshDiags.HasError() {
			diagnostics.Append(refreshDiags...)
			continue
		}

		// If the machine is not found, return ok as we cannot remove something that does not exist
		if currentVm == nil {
			// We are waiting for half of the time to wait for the orchestrator to update
			time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
			currentVm, refreshDiags := apiclient.GetVm(ctx, hostConfig, vmIdOrName)
			if refreshDiags.HasError() {
				diagnostics.Append(refreshDiags...)
				continue
			}
			if currentVm == nil {
				diagnostics = diag.Diagnostics{}
				break
			}
		}

		tflog.Info(ctx, "Checking if "+currentVm.Name+" is running")
		// making sure the VM is stopped before removing it
		if _, ensureStopped := EnsureMachineStopped(ctx, hostConfig, currentVm); ensureStopped.HasError() {
			diagnostics.Append(ensureStopped...)
			continue
		}

		// Refresh the state of the machine
		refreshedVm, refreshedVmDiag := apiclient.GetVm(ctx, hostConfig, currentVm.ID)
		if refreshedVmDiag.HasError() {
			diagnostics.Append(refreshedVmDiag...)
			continue
		}

		if refreshedVm == nil {
			// We are waiting for half of the time to wait for the orchestrator to update
			time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
			refreshedVm, refreshDiags := apiclient.GetVm(ctx, hostConfig, vmIdOrName)
			if refreshDiags.HasError() {
				diagnostics.Append(refreshDiags...)
				continue
			}
			if refreshedVm == nil {
				diagnostics = diag.Diagnostics{}
				break
			}
		}

		// The machine is stopped, we can remove it
		if refreshedVm.State == "stopped" {
			tflog.Info(ctx, "Machine "+currentVm.Name+" is stopped, removing it")
			if removeDiag := apiclient.DeleteVm(ctx, hostConfig, refreshedVm.ID); removeDiag.HasError() {
				diagnostics.Append(removeDiag...)
				continue
			}
			// The machine has been removed, break out of the loop
			break
		}

		time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
	}

	return diagnostics
}
