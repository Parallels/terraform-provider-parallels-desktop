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

func PullCatalog(ctx context.Context, config HostConfig, request apimodels.PullCatalogRequest) (*apimodels.PullCatalogResponse, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	var response apimodels.PullCatalogResponse
	urlHost := helpers.GetHostUrl(config.Host)
	url := helpers.GetHostApiVersionedBaseUrl(urlHost) + "/catalog/pull"

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostic
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.PutDataToClient(ctx, url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error getting vms: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostic.AddError("There was an error pulling the catalog "+request.CatalogId, err.Error())
		return nil, diagnostic
	}

	tflog.Info(ctx, "Pull catalog "+response.ID)

	return &response, diagnostic
}
