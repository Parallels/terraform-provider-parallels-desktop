package apiclient

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func DeleteApiKey(ctx context.Context, config HostConfig, apiKeyId string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if apiKeyId == "" {
		diagnostic.AddError("There was an error deleting the api key", "api key id is empty")
		return diagnostic
	}

	url := fmt.Sprintf("%s/auth/api_keys/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), apiKeyId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return diagnostic
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.DeleteDataFromClient(url, nil, auth, nil); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				tflog.Info(ctx, "Api Key "+apiKeyId+" not found")
				return diagnostic
			}
		}
		diagnostic.AddError("There was an error deleting the api key", err.Error())
		return diagnostic
	}

	tflog.Info(ctx, "Deleted api key "+apiKeyId)

	return diagnostic
}
