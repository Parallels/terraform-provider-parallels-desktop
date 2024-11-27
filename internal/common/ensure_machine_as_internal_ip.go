package common

import (
	"context"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"terraform-provider-parallels-desktop/internal/constants"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func EnsureMachineHasInternalIp(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine) (*apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}

	refreshVm, ensureRunningDiag := EnsureMachineRunning(ctx, hostConfig, vm)
	if ensureRunningDiag.HasError() {
		diagnostics.Append(ensureRunningDiag...)
		return nil, diagnostics
	}
	if refreshVm == nil {
		diagnostics.AddError("There was an error getting the vm", "vm is nil")
		return nil, diagnostics
	}

	maxRetries := constants.DEFAULT_OPERATION_MAX_RETRY_COUNT
	retryCount := 0
	for {
		diagnostics = diag.Diagnostics{}
		retryCount += 1
		// We have run out of retries, add an error and break out of the loop
		if retryCount >= maxRetries {
			diagnostics.AddError("Error in Internal IP", "We could not get the internal IP of the machine")
			break
		}

		if refreshVm.InternalIpAddress == "" || refreshVm.InternalIpAddress == "-" {
			updatedVm, checkVmDiag := apiclient.GetVm(ctx, hostConfig, refreshVm.ID)
			if checkVmDiag.HasError() {
				diagnostics.Append(checkVmDiag...)
				tflog.Error(ctx, "Error getting vm Internal IP")
				continue
			}
			if updatedVm == nil {
				diagnostics.AddError("Error in Internal IP", "VM not found")
				tflog.Error(ctx, "Error getting vm Internal IP, VM not found")
				continue
			}

			// If we have the internal IP, break out of the loop
			if updatedVm.InternalIpAddress != "" && updatedVm.InternalIpAddress != "-" {
				tflog.Info(ctx, "Machine "+updatedVm.Name+" is running")
				diagnostics = diag.Diagnostics{}
				refreshVm = updatedVm
				break
			}

			time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
		} else {
			break
		}
	}

	return refreshVm, diagnostics
}
