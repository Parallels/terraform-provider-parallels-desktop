package apiclient

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func DeleteClaim(ctx context.Context, config HostConfig, claimId string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if claimId == "" {
		diagnostic.AddError("There was an error deleting the claim", "claim id is empty")
		return diagnostic
	}

	url := fmt.Sprintf("%s/auth/claims/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), claimId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return diagnostic
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.DeleteDataFromClient(url, nil, auth, nil); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				tflog.Info(ctx, "Claim "+claimId+" not found")
				return diagnostic
			}
		}
		diagnostic.AddError("There was an error deleting the claim", err.Error())
		return diagnostic
	}

	tflog.Info(ctx, "Deleted claim "+claimId)

	return diagnostic
}
