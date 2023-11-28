package apiclient

import (
	"context"
	"fmt"
	"strconv"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/constants"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func GetPackerTemplates(ctx context.Context, config HostConfig, filterField, filterValue string) ([]apimodels.PackerTemplate, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	response := make([]apimodels.PackerTemplate, 0)
	urlHost := helpers.GetHostUrl(config.Host)
	url := fmt.Sprintf("%s/templates/packer", helpers.GetHostApiVersionedBaseUrl(urlHost))

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization)
	if err != nil {
		diagnostic.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostic
	}

	var filter map[string]string
	if filterField != "" && filterValue != "" {
		filter = map[string]string{
			constants.FILTER_HEADER: fmt.Sprintf("%s=%s", filterField, filterValue),
		}
	}

	client := helpers.NewHttpCaller(ctx)
	if clientResponse, err := client.GetDataFromClient(url, &filter, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error getting packer templates: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostic.AddError("There was an error getting the packer templates", err.Error())
		return nil, diagnostic
	}

	tflog.Info(ctx, "Got "+strconv.Itoa(len(response))+" packer templates")

	return response, diagnostic
}
