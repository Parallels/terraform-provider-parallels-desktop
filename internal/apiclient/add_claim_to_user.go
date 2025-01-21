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

func AddClaimToUser(ctx context.Context, config HostConfig, userId string, claim string) (*apimodels.ClaimRoleResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ClaimRoleResponse
	if claim == "" {
		diagnostics.AddError("There was an error adding claim to user", "claim is empty")
		return nil, diagnostics
	}

	tflog.Info(ctx, "Adding Claim "+claim+" to User "+userId)
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/auth/users/%s/claims", helpers.GetHostApiVersionedBaseUrl(urlHost), userId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	request := apimodels.AddUserClaimRoleRequest{
		Name: claim,
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.PostDataToClient(ctx, url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error adding claim to user: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error adding claim to user", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
