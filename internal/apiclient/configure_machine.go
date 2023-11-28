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

func ConfigureMachine(ctx context.Context, config HostConfig, machineId string, configSet *apimodels.VmConfigRequest) (*apimodels.VmConfigResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	tflog.Debug(ctx, fmt.Sprintf("Configuring machine %v with configSet", *configSet))

	client := helpers.NewHttpCaller(ctx)
	var response apimodels.VmConfigResponse
	if clientResponse, err := client.PutDataToClient(fmt.Sprintf("%s/machines/%s/set", helpers.GetHostApiVersionedBaseUrl(urlHost), machineId), nil, configSet, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error configuring vm: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error configuring the machine", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
