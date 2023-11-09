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

func ExecuteScript(ctx context.Context, config HostConfig, machineId string, script string) (*apimodels.VmExecuteCommandResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/machines/%s/execute", helpers.GetHostApiBaseUrl(urlHost), machineId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}
	request := apimodels.VmExecuteCommandRequest{
		Command: script,
	}

	client := helpers.NewHttpCaller(ctx)
	var response apimodels.VmExecuteCommandResponse
	if clientResponse, err := client.PostDataToClient(url, nil, request, auth, &response); err != nil {
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
