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

func EnsureMachineRunning(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine) (*apimodels.VirtualMachine, diag.Diagnostics) {
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
			diagnostics.AddError("Error starting vm", "Could not verify the state of the machine after starting, retry count exceeded")
			break
		}

		if currentVm.State != "running" {
			tflog.Info(ctx, "Machine "+currentVm.Name+" is not running, starting it"+fmt.Sprintf("[%v/%v]", retryCount, maxRetries))
			op := apiclient.MachineStateOpStart
			if currentVm.State == "suspended" || currentVm.State == "paused" {
				op = apiclient.MachineStateOpResume
			}
			stateResult, stateDiag := apiclient.SetMachineState(ctx, hostConfig, currentVm.ID, op)
			if stateDiag.HasError() {
				diagnostics.Append(stateDiag...)
				continue
			}

			if !stateResult {
				diagnostics.AddError("Error starting vm", "Could not set the state of the machine to running")
				continue
			}

			tflog.Info(ctx, "Checking if "+currentVm.Name+" is running")

			refreshedVm, refreshedVmDiag := apiclient.GetVm(ctx, hostConfig, currentVm.ID)
			if refreshedVmDiag.HasError() {
				diagnostics.Append(refreshedVmDiag...)
				continue
			}
			if refreshedVm == nil {
				diagnostics.AddError("Error starting vm", "Could not verify the state of the machine after starting, retry count exceeded")
				continue
			}

			// The machine is running, lets check if we have the tools initialized
			if refreshedVm.State == "running" {
				echoHelloCommand := apimodels.PostScriptItem{
					Command:          "echo 'I am running'",
					VirtualMachineId: refreshedVm.ID,
				}

				// Only breaking out of the loop if the script executes successfully
				if _, execDiag := apiclient.ExecuteScript(ctx, hostConfig, echoHelloCommand); !execDiag.HasError() {
					tflog.Info(ctx, "Machine "+currentVm.Name+" is running")
					diagnostics = diag.Diagnostics{}
					currentVm = refreshedVm
					break
				}
			}

			time.Sleep(constants.DEFAULT_OPERATION_RETRY_INTERVAL_IN_SECONDS * time.Second)
		} else {
			break
		}
	}

	return currentVm, diagnostics
}
