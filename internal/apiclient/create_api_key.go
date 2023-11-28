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

func CreateApiKey(ctx context.Context, config HostConfig, request apimodels.ApiKeyRequest) (*apimodels.ApiKeyResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ApiKeyResponse

	if err := request.Validate(); err != nil {
		diagnostics.AddError("There was an error validating the api key request", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, "Creating API Key "+request.Name)
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/auth/api_keys", helpers.GetHostApiVersionedBaseUrl(urlHost))

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.PostDataToClient(url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error adding api key: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error creating the api key", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
