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

func GetOrchestratorHost(ctx context.Context, config HostConfig, hostId string) (*apimodels.OrchestratorHost, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response apimodels.OrchestratorHost
	urlHost := helpers.GetHostUrl(config.Host)
	if hostId == "" {
		diagnostics.AddError("There was an error getting the orchestrator host", "orchestratorHostId is empty")
		return nil, diagnostics
	}

	url := fmt.Sprintf("%s/orchestrator/hosts/%s", helpers.GetHostApiVersionedBaseUrl(urlHost), hostId)

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostics
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting orchestrator hosts: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error getting the orchestrator hosts", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, "Got orchestrator "+response.Description)

	return &response, diagnostics
}

func GetOrchestratorHosts(ctx context.Context, config HostConfig) ([]apimodels.OrchestratorHost, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	var response []apimodels.OrchestratorHost
	urlHost := helpers.GetHostUrl(config.Host)

	url := fmt.Sprintf("%s/orchestrator/hosts", helpers.GetHostApiVersionedBaseUrl(urlHost))

	auth, err := authenticator.GetAuthenticator(ctx, urlHost, config.License, config.Authorization, config.DisableTlsValidation)
	if err != nil {
		diagnostics.AddError("There was an error getting the authenticator", err.Error())
		return nil, diagnostics
	}

	client := helpers.NewHttpCaller(ctx, config.DisableTlsValidation)
	if clientResponse, err := client.GetDataFromClient(url, nil, auth, &response); err != nil {
		if clientResponse != nil && clientResponse.ApiError != nil {
			if clientResponse.ApiError.Code == 404 {
				return nil, diagnostics
			}
			tflog.Error(ctx, fmt.Sprintf("Error getting orchestrator hosts: %v, api message: %s", err, clientResponse.ApiError.Message))
		}
		diagnostics.AddError("There was an error getting the orchestrator hosts", err.Error())
		return nil, diagnostics
	}

	tflog.Info(ctx, fmt.Sprintf("Got %v orchestrators ", len(response)))

	return response, diagnostics
}
