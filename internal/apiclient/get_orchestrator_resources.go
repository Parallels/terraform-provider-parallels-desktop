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

func GetOrchestratorResources(ctx context.Context, config HostConfig) ([]*apimodels.SystemUsageResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response []*apimodels.SystemUsageResponse
	urlHost := helpers.GetHostUrl(config.Host)

	url := fmt.Sprintf("%s/orchestrator/overview/resources", helpers.GetHostApiVersionedBaseUrl(urlHost))

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error orchestrator resources: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error getting orchestrator resources", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, "Got Orchestrator resources ")

	return response, diagnostics
}
