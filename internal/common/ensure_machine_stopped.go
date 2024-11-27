package common

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/constants"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func EnsureMachineStopped(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine) (*apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}

	currentVm, refreshDiags := apiclient.GetVm(ctx, hostConfig, vm.ID)
	if refreshDiags.HasError() {
		diagnostics.Append(refreshDiags...)
		return vm, diagnostics
	}
	if currentVm == nil {
		diagnostics.AddError("There was an error getting the vm", "vm is nil")
		return vm, diagnostics
	}

	maxRetries := constants.DEFAULT_OPERATION_MAX_RETRY_COUNT
	retryCount := 0
	for {
		diagnostics = diag.Diagnostics{}
		retryCount += 1

		// We have run out of retries, add an error and break out of the loop
		if retryCount >= maxRetries {
			diagnostics.AddError("error stopping vm", "error stopping vm")
			break
		}

		if currentVm.State != "stopped" {
			tflog.Info(ctx, "Machine "+currentVm.Name+" is not stopped, stopping it"+fmt.Sprintf("[%v/%v]", retryCount, maxRetries))
			result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, currentVm.ID, apiclient.MachineStateOpStop)
			if stateDiag.HasError() {
				diagnostics.Append(stateDiag...)
				continue
			}

			if !result {
				diagnostics.AddError("error stopping vm", "Could not set the state of the machine to stopped")
				continue
			}

			tflog.Info(ctx, "Checking if "+currentVm.Name+" is stopped")

			refreshedVm, refreshedVmDiag := apiclient.GetVm(ctx, hostConfig, currentVm.ID)
			if refreshedVmDiag.HasError() {
				diagnostics.Append(refreshedVmDiag...)
				continue
			}
			if refreshedVm == nil {
				diagnostics.AddError("error stopping vm", "Could not verify the state of the machine after stopping, vm is nil")
				continue
			}

			// All if good, break out of the loop
			if refreshedVm.State == "stopped" {
				tflog.Info(ctx, "Machine "+currentVm.Name+" is stopped")
				diagnostics = diag.Diagnostics{}
				currentVm = refreshedVm
				break
			}

			time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
		} else {
			break
		}
	}

	return currentVm, diagnostics
}
