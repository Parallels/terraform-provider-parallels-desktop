package apiclient

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/constants"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func GetVms(ctx context.Context, config HostConfig, filterField, filterValue string) ([]apimodels.VirtualMachine, diag.Diagnostics) {
	diagnostic := diag.Diagnostics{}
	response := make([]apimodels.VirtualMachine, 0)
	urlHost := helpers.GetHostUrl(config.Host)
	filterValue = strings.ReplaceAll(filterValue, "\"", "")
	var url string

	if config.IsOrchestrator {
		url = helpers.GetHostApiVersionedBaseUrl(urlHost) + "/orchestrator/machines/"
	} else {
		url = helpers.GetHostApiVersionedBaseUrl(urlHost) + "/machines/"
	}

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
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

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(ctx, url, &filter, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			tflog.Error(ctx, fmt.Sprintf("Error getting vms: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostic.AddError("There was an error getting the vms", err.Error())
		return nil, diagnostic
	}

	tflog.Info(ctx, "Got "+strconv.Itoa(len(response))+" machines")

	return response, diagnostic
}
