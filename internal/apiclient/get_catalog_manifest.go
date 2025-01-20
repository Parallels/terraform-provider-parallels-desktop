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

func GetCatalogManifest(ctx context.Context, config HostConfig, catalogId string, version string, architecture string) (*apimodels.CatalogManifest, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response *apimodels.CatalogManifest
	urlHost := helpers.GetHostUrl(config.Host)
	if catalogId == "" {
		diagnostics.AddError("There was an error getting the catalog manifest", "catalogId is empty")
		return nil, diagnostics
	}
	if version == "" {
		diagnostics.AddError("There was an error getting the catalog manifest", "version is empty")
		return nil, diagnostics
	}
	if architecture == "" {
		architecture = "arm64"
	}

	url := fmt.Sprintf("%s/catalog/%s/%s/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), catalogId, version, architecture)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(ctx, url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostics
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting catalog manifest: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error getting the catalog manifest", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, "Got catalog manifest ")

	return response, diagnostics
}
