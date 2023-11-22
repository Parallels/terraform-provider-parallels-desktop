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

func GetRole(ctx context.Context, config HostConfig, roleId string) (*apimodels.ClaimRoleResponse, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	var response apimodels.ClaimRoleResponse
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/auth/roles/%s", helpers.GetHostApiBaseUrl(urlHost), roleId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostic
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostic
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting roles: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostic.AddError("There was an error getting the roles", err.Error())
		return nil, diagnostic
	}

	tflog.Info(ctx, "Got Role "+response.ID)

	return &response, diagnostic
}
