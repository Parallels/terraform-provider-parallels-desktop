package apiclient

import (
	"context"
	"fmt"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func DeleteUser(ctx context.Context, config HostConfig, userId string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	urlHost := helpers.GetHostUrl(config.Host)
	if userId == "" {
		diagnostic.AddError("There was an error deleting the user", "user id is empty")
		return diagnostic
	}

	url := fmt.Sprintf("%s/auth/users/%s", helpers.GetHostApiBaseUrl(urlHost), userId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return diagnostic
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.DeleteDataFromClient(url, nil, auth, nil); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				tflog.Info(ctx, "User "+userId+" not found")
				return diagnostic
			}
		}
		diagnostic.AddError("There was an error deleting the user", err.Error())
		return diagnostic
	}

	tflog.Info(ctx, "Deleted user "+userId)

	return diagnostic
}
