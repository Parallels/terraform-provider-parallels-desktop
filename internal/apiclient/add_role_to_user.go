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

func AddRoleToUser(ctx context.Context, config HostConfig, userId string, role string) (*apimodels.ClaimRoleResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ClaimRoleResponse
	if role == "" {
		diagnostics.AddError("There was an error adding role to user", "role is empty")
		return nil, diagnostics
	}

	tflog.Info(ctx, "Adding Role "+role+" to User "+userId)
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/auth/users/%s/roles", helpers.GetHostApiBaseUrl(urlHost), userId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	request := apimodels.AddUserClaimRoleRequest{
		Name: role,
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.PostDataToClient(url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error adding role to user: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error adding role to user", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
