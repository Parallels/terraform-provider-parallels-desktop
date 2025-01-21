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

func CreateReverseProxyHost(ctx context.Context, config HostConfig, request apimodels.ReverseProxyHost) (*apimodels.ReverseProxyHost, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ReverseProxyHost

	tflog.Info(ctx, "Creating reverse proxy host "+request.Host+" with port "+request.Port)
	urlHost := helpers.GetHostUrl(config.Host)
	var url string
	if config.IsOrchestrator {
		url = fmt.Sprintf("%s/orchestrator/hosts/%s/reverse-proxy/hosts", helpers.GetHostApiVersionedBaseUrl(urlHost), config.HostId)
	} else {
		url = helpers.GetHostApiVersionedBaseUrl(urlHost) + "/reverse-proxy/hosts"
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.PostDataToClient(ctx, url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error creating reverse proxy: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error creating the reverse proxy", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
