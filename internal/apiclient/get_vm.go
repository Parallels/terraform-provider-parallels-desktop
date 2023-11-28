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

func GetVm(ctx context.Context, config HostConfig, machineId string) (*apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.VirtualMachine
	urlHost := helpers.GetHostUrl(config.Host)
	if machineId == "" {
		diagnostics.AddError("There was an error getting the vm", "machineId is empty")
		return nil, diagnostics
	}

	url := fmt.Sprintf("%s/machines/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), machineId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostics
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting vms: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error getting the vms", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, "Got machine "+response.Name)

	return &response, diagnostics
}
