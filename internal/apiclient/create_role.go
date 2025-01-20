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

func CreateRole(ctx context.Context, config HostConfig, roleName string) (*apimodels.ClaimRoleResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ClaimRoleResponse
	if roleName == "" {
		diagnostics.AddError("There was an error validating the role request", "role name is empty")
		return nil, diagnostics
	}

	request := apimodels.AddUserClaimRoleRequest{
		Name: roleName,
	}

	tflog.Info(ctx, "Creating Role "+request.Name)
	urlHost := helpers.GetHostUrl(config.Host)
	url := helpers.GetHostApiVersionedBaseUrl(urlHost) + "/auth/roles"

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.PostDataToClient(ctx, url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error creating role: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error creating the role", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
