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

func GetReverseProxyHost(ctx context.Context, config HostConfig, host string) (*apimodels.ReverseProxyHost, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if host == "" {
		diagnostic.AddError("There was an error getting the reverse proxy host", "host is empty")
		return nil, diagnostic
	}

	var url string
	if config.IsOrchestrator {
		url = fmt.Sprintf("%s/orchestrator/hosts/%s/reverse-proxy/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), config.HostId, host)
	} else {
		url = fmt.Sprintf("%s/reverse-proxy/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), host)
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostic
	}

	var response apimodels.ReverseProxyHost
	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostic
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting claims: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostic.AddError("There was an error getting the claims", err.Error())
		return nil, diagnostic
	}

	tflog.Info(ctx, "Got the reverse proxy host "+host)

	return &response, diagnostic
}
