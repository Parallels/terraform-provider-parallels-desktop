package apiclient

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func UnregisterWithOrchestrator(ctx context.Context, config HostConfig, hostId string) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/orchestrator/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), hostId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.DeleteDataFromClient(ctx, url, nil, auth, nil); err != nil {
		if clientResponse != nil {
			if clientResponse.StatusCode == http.StatusNotFound {
				tflog.Warn(ctx, fmt.Sprintf("Host %s not found", hostId))
				return diagnostics
			}
			if clientResponse.ApiError != nil {
				tflog.Error(ctx, fmt.Sprintf("Error un-registering host: %v, api message: %s", err, clientResponse.ApiError.Message))
			}

			diagnostics.AddError("There was an error un-registering host", err.Error())
			return diagnostics
		}
	}

	return diagnostics
}
