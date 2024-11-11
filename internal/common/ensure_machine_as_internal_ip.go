package common

import (
	"context"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

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

	maxRetries := 10
	retryCount := 0
	for {
		diagnostics = diag.Diagnostics{}
		retryCount += 1
		if refreshVm.InternalIpAddress == "" || refreshVm.InternalIpAddress == "-" {
			updatedVm, checkVmDiag := apiclient.GetVm(ctx, hostConfig, refreshVm.ID)
			if checkVmDiag.HasError() {
				diagnostics.Append(checkVmDiag...)
			}

			// If we have the internal IP, break out of the loop
			if updatedVm.InternalIpAddress != "" && updatedVm.InternalIpAddress != "-" {
				tflog.Info(ctx, "Machine "+updatedVm.Name+" is running")
				diagnostics = diag.Diagnostics{}
				refreshVm = updatedVm
				break
			}

			// We have run out of retries, add an error and break out of the loop
			if retryCount >= maxRetries {
				diagnostics.AddError("error getting vm Internal IP", "We could not get the internal IP of the machine")
				break
			}

			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	return refreshVm, diagnostics
}
