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

func EnsureMachineHasInternalIp(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine, timeout time.Duration) (*apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}

	if vm == nil {
		diagnostics.AddError("There was an error getting the vm", "vm is nil")
		return nil, diagnostics
	}

	refreshVm, ensureRunningDiag := EnsureMachineRunning(ctx, hostConfig, vm)
	if ensureRunningDiag.HasError() {
		diagnostics.Append(ensureRunningDiag...)
		return nil, diagnostics
	}
	if refreshVm == nil {
		diagnostics.AddError("There was an error getting the vm", "vm is nil")
		return nil, diagnostics
	}

	if timeout <= 0 {
		timeout = DefaultVmReadyTimeout
	}
	deadline := time.Now().Add(timeout)
	for {
		diagnostics = diag.Diagnostics{}
		// We have run out of time, add an error and break out of the loop
		if time.Now().After(deadline) {
			diagnostics.AddError("Error in Internal IP", fmt.Sprintf("We could not get the internal IP of the machine within %s", timeout))
			break
		}

		if refreshVm.InternalIpAddress == "" || refreshVm.InternalIpAddress == "-" {
			updatedVm, checkVmDiag := apiclient.GetVm(ctx, hostConfig, refreshVm.ID)
			if checkVmDiag.HasError() {
				diagnostics.Append(checkVmDiag...)
				tflog.Error(ctx, "Error getting vm Internal IP")
				time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
				continue
			}
			if updatedVm == nil {
				diagnostics.AddError("Error in Internal IP", "VM not found")
				tflog.Error(ctx, "Error getting vm Internal IP, VM not found")
				time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
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
