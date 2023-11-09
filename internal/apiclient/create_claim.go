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

func CreateClaim(ctx context.Context, config HostConfig, claimName string) (*apimodels.ClaimRoleResponse, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.ClaimRoleResponse
	if claimName == "" {
		diagnostics.AddError("There was an error validating the claim request", "claim name is empty")
		return nil, diagnostics
	}

	request := apimodels.AddUserClaimRoleRequest{
		Name: claimName,
	}

	tflog.Info(ctx, "Creating Claim "+request.Name)
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/auth/claims", helpers.GetHostApiBaseUrl(urlHost))

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.PostDataToClient(url, nil, request, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error creating claim: %v, api message: %s", err, clientResponse.ApiError.Message))
		}

		diagnostics.AddError("There was an error creating the claim", err.Error())
		return nil, diagnostics
	}

	return &response, diagnostics
}
