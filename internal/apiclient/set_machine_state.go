package apiclient

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type MachineStateOp string

const (
	MachineStateOpStart   MachineStateOp = "start"
	MachineStateOpStop    MachineStateOp = "stop"
	MachineStateOpRestart MachineStateOp = "restart"
	MachineStateOpPause   MachineStateOp = "pause"
	MachineStateOpResume  MachineStateOp = "resume"
	MachineStateOpSuspend MachineStateOp = "suspend"
)

func SetMachineState(ctx context.Context, config HostConfig, machineId string, op MachineStateOp) (bool, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)

	var url string
	if config.IsOrchestrator {
		url = fmt.Sprintf("%s/orchestrator/machines/%s/set", helpers.GetHostApiVersionedBaseUrl(urlHost), machineId)
	} else {
		url = fmt.Sprintf("%s/machines/%s/set", helpers.GetHostApiVersionedBaseUrl(urlHost), machineId)
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return false, diagnostics
	}

	vm, diag := GetVm(ctx, config, machineId)
	if diag.HasError() {
		diagnostics = append(diagnostics, diag...)
		return false, diagnostics
	}

	if vm.State == string(op) {
		tflog.Info(ctx, fmt.Sprintf("Machine %s is already in state %s", machineId, op))
		return true, diagnostics
	}

	configSet := apimodels.NewVmConfigRequest(vm.User)
	setOp := apimodels.NewVmConfigRequestOperation(configSet)
	setOp.WithGroup("state")
	setOp.WithOperation(string(op))
	setOp.Append()

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	var response apimodels.VmConfigResponse
	if clientResponse, err := client.PutDataToClient(ctx, url, nil, configSet, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error configuring vm: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error setting the machine state", err.Error())
		return false, diagnostics
	}

	return true, diagnostics
}
