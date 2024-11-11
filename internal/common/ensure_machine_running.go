package common

import (
	"context"
	"fmt"
	"time"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func EnsureMachineRunning(ctx context.Context, hostConfig apiclient.HostConfig, vm *apimodels.VirtualMachine) (*apimodels.VirtualMachine, diag.Diagnostics) {
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
		if returnVm.State != "running" {
			tflog.Info(ctx, "Machine "+returnVm.Name+" is not running, starting it"+fmt.Sprintf("[%v/%v]", retryCount, maxRetries))
			result, stateDiag := apiclient.SetMachineState(ctx, hostConfig, returnVm.ID, apiclient.MachineStateOpStart)
			if stateDiag.HasError() {
				diagnostics.Append(stateDiag...)
			}

			if !result {
				diagnostics.AddError("error starting vm", "error starting vm")
			}

			tflog.Info(ctx, "Checking if "+returnVm.Name+" is running")

			updatedVm, checkVmDiag := apiclient.GetVm(ctx, hostConfig, returnVm.ID)
			if checkVmDiag.HasError() {
				diagnostics.Append(checkVmDiag...)
			}

			// The machine is running, lets check if we have the tools initialized
			if updatedVm.State == "running" {
				echoHelloCommand := apimodels.PostScriptItem{
					Command:          "echo 'I am running'",
					VirtualMachineId: updatedVm.ID,
				}

				// Only breaking out of the loop if the script executes successfully
				if _, execDiag := apiclient.ExecuteScript(ctx, hostConfig, echoHelloCommand); !execDiag.HasError() {
					tflog.Info(ctx, "Machine "+returnVm.Name+" is running")
					diagnostics = diag.Diagnostics{}
					returnVm = updatedVm
					break
				}
			}

			// We have run out of retries, add an error and break out of the loop
			if retryCount >= maxRetries {
				diagnostics.AddError("error starting vm", "error starting vm")
				break
			}

			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	return returnVm, diagnostics
}
