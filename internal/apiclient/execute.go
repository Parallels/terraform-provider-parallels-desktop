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

func ExecuteScript(ctx context.Context, config HostConfig, r apimodels.PostScriptItem) (*apimodels.VmExecuteCommandResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	var url string
	if config.IsOrchestrator {
		url = fmt.Sprintf("%s/orchestrator/machines/%s/execute", helpers.GetHostApiVersionedBaseUrl(urlHost), r.VirtualMachineId)
	} else {
		url = fmt.Sprintf("%s/machines/%s/execute", helpers.GetHostApiVersionedBaseUrl(urlHost), r.VirtualMachineId)
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	request := apimodels.VmExecuteCommandRequest{
		Command:              r.Command,
		EnvironmentVariables: r.EnvironmentVariables,
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	var response apimodels.VmExecuteCommandResponse
	if clientResponse, err := client.PutDataToClient(url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error executing script on vm: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error executing the script1", err.Error())
		return nil, diagnostics
	}

	if response.ExitCode != 0 {
		diagnostics.AddError("There was an error executing the script", fmt.Sprintf("Err: %v, Exit code: %v", response.Stderr, response.ExitCode))
		return nil, diagnostics
	}

	return &response, diagnostics
}
