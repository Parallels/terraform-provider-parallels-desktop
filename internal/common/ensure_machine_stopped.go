package common

import (
	"context"
	"fmt"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func EnsureMachineStopped(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine) (*apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}

	returnVm, refreshDiags := apiclient.GetVm(ctx, hostConfig, vm.ID)
	if refreshDiags.HasError() {
		diagnostics.Append(refreshDiags...)
		return vm, diagnostics
	}

	maxRetries := 10
	retryCount := 0
	for {
		diagnostics = diag.Diagnostics{}
		retryCount += 1
		if returnVm.State != "stopped" {
			tflog.Info(ctx, "Machine "+returnVm.Name+" is not stopped, stopping it"+fmt.Sprintf("[%v/%v]", retryCount, maxRetries))
			result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, returnVm.ID, apiclient.MachineStateOpStop)
			if stateDiag.HasError() {
				diagnostics.Append(stateDiag...)
			}

			if !result {
				diagnostics.AddError("error stopping vm", "error stopping vm")
			}

			tflog.Info(ctx, "Checking if "+returnVm.Name+" is stopped")

			updatedVm, checkVmDiag := apiclient.GetVm(ctx, hostConfig, returnVm.ID)
			if checkVmDiag.HasError() {
				diagnostics.Append(checkVmDiag...)
			}

			// All if good, break out of the loop
			if updatedVm.State == "stopped" {
				tflog.Info(ctx, "Machine "+returnVm.Name+" is stopped")
				diagnostics = diag.Diagnostics{}
				returnVm = updatedVm
				break
			}

			// We have run out of retries, add an error and break out of the loop
			if retryCount >= maxRetries {
				diagnostics.AddError("error stopping vm", "error stopping vm")
				break
			}

			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	return returnVm, diagnostics
}
