package orchestrator

import (
	"context"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func RegisterWithHost(context context.Context, plan OrchestratorRegistration) (string, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}
	if updateDiags := UpdateFromDetails(context, &plan); updateDiags.HasError() {
		diagnostics.Append(updateDiags...)
		return "", diagnostics
	}

	if plan.Host.ValueString() == "" {
		diagnostics.AddError("Error registering with orchestrator", "we cannot register with orchestrator if we are on localhost")
		return "", diagnostics
	}
	if plan.Orchestrator == nil || plan.Orchestrator.Host.ValueString() == "" {
		diagnostics.AddError("Error registering with orchestrator", "we cannot register with orchestrator, host is empty")
		return "", diagnostics
	}
	if plan.Orchestrator.UseAuthentication == nil {
		diagnostics.AddError("Error registering with orchestrator", "we cannot register with orchestrator, authentication is empty")
		return "", diagnostics
	}
	if plan.HostCredentials == nil {
		diagnostics.AddError("Error registering with orchestrator", "we cannot register with orchestrator, authentication is empty")
		return "", diagnostics
	}

	host := plan.Host.ValueString()
	if plan.Schema.ValueString() != "" {
		host = plan.Schema.ValueString() + "://" + host
	}
	if plan.Port.ValueString() != "" {
		host = host + ":" + plan.Port.ValueString()
	}

	orchestratorRequest := apimodels.OrchestratorHostRequest{
		Host:        host,
		Tags:        plan.Tags,
		Description: plan.Description.ValueString(),
	}

	if plan.HostCredentials != nil {
		orchestratorRequest.Authentication = &apimodels.OrchestratorAuthentication{}
		orchestratorRequest.Authentication.Username = plan.HostCredentials.Username.ValueString()
		orchestratorRequest.Authentication.Password = plan.HostCredentials.Password.ValueString()
		orchestratorRequest.Authentication.ApiKey = plan.HostCredentials.ApiKey.ValueString()
	}

	hostConfig := apiclient.HostConfig{
		Host:          plan.Orchestrator.GetHost(),
		Authorization: plan.Orchestrator.UseAuthentication,
	}

	response, diag := apiclient.RegisterWithOrchestrator(context, hostConfig, orchestratorRequest)
	if diag.HasError() {
		diagnostics.Append(diag...)
		return "", diagnostics
	}

	if response != nil {
		return response.ID, diagnostics
	}

	return "", diagnostics
}

func IsAlreadyRegistered(context context.Context, data OrchestratorRegistration) (bool, diag.Diagnostics) {
	diagnostics := diag.Diagnostics{}

	hostConfig := apiclient.HostConfig{
		Host: data.Orchestrator.GetHost(),
		Authorization: &authenticator.Authentication{
			Username: data.Orchestrator.UseAuthentication.Username,
			Password: data.Orchestrator.UseAuthentication.Password,
			ApiKey:   data.Orchestrator.UseAuthentication.ApiKey,
		},
	}

	currentHostId := data.HostId.ValueString()
	currentHostUrl := helpers.GetHostApiBaseUrl(data.GetHost())
	response, _ := apiclient.GetOrchestratorHost(context, hostConfig, data.HostId.ValueString())
	if response == nil {
		return false, diagnostics
	}

	if currentHostId == response.ID || currentHostUrl == response.Host {
		return true, diagnostics
	}

	return false, diagnostics
}

func UnregisterWithHost(context context.Context, data OrchestratorRegistration) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}

	hostConfig := apiclient.HostConfig{
		Host: data.Orchestrator.GetHost(),
		Authorization: &authenticator.Authentication{
			Username: data.Orchestrator.UseAuthentication.Username,
			Password: data.Orchestrator.UseAuthentication.Password,
			ApiKey:   data.Orchestrator.UseAuthentication.ApiKey,
		},
	}

	_ = UpdateFromDetails(context, &data)
	if data.HostId.ValueString() != "" {
		diag := apiclient.UnregisterWithOrchestrator(context, hostConfig, data.HostId.ValueString())
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
	}

	return diagnostics
}

func UpdateFromDetails(context context.Context, data *OrchestratorRegistration) diag.Diagnostics {
	diagnostics := diag.Diagnostics{}
	if data.Orchestrator == nil {
		return diagnostics
	}

	if data.Host.ValueString() == "" {
		if data.Orchestrator.Host.ValueString() == "" {
			diagnostics.AddError("Host is required", "")
			return diagnostics
		} else {
			data.Host = data.Orchestrator.Host
		}
	}

	if data.Port.ValueString() == "" {
		if data.Orchestrator.Port.ValueString() == "" {
			diagnostics.AddError("Port is required", "")
		} else {
			data.Port = data.Orchestrator.Port
		}
	}

	if data.Schema.ValueString() == "" {
		if data.Orchestrator.Schema.ValueString() == "" {
			diagnostics.AddError("Description is required", "")
		} else {
			data.Schema = data.Orchestrator.Schema
		}
	}

	if data.HostCredentials == nil {
		data.HostCredentials = data.Orchestrator.UseAuthentication
	}

	if data.HostCredentials.ApiKey.ValueString() == "" {
		if data.Orchestrator.UseAuthentication.ApiKey.ValueString() != "" {
			data.HostCredentials.ApiKey = data.Orchestrator.UseAuthentication.ApiKey
		}
	}

	if data.HostCredentials.Username.ValueString() == "" {
		if data.Orchestrator.UseAuthentication.Username.ValueString() != "" {
			data.HostCredentials.Username = data.Orchestrator.UseAuthentication.Username
		}
	}

	if data.HostCredentials.Password.ValueString() == "" {
		if data.Orchestrator.UseAuthentication.Password.ValueString() != "" {
			data.HostCredentials.Password = data.Orchestrator.UseAuthentication.Password
		}
	}

	if data.HostCredentials.ApiKey.ValueString() == "" && data.HostCredentials.Username.ValueString() == "" && data.HostCredentials.Password.ValueString() == "" {
		diagnostics.AddError("Host credentials are required", "")
	}

	return diagnostics
}

func HasChanges(context context.Context, planState, currentState *OrchestratorRegistration) bool {
	if planState == nil && currentState == nil {
		return false
	}
	if planState != nil && currentState == nil {
		return true
	}
	if planState == nil && currentState != nil {
		return true
	}

	if planState.Host != currentState.Host {
		return true
	}
	if planState.Port != currentState.Port {
		return true
	}
	if planState.Description != currentState.Description {
		return true
	}
	if planState.Schema != currentState.Schema {
		return true
	}
	if len(planState.Tags) != len(currentState.Tags) {
		return true
	}
	for i, tag := range planState.Tags {
		if tag != currentState.Tags[i] {
			return true
		}
	}

	if planState.HostCredentials != nil && currentState.HostCredentials == nil {
		return true
	}
	if planState.HostCredentials == nil && currentState.HostCredentials != nil {
		return true
	}
	if planState.HostCredentials != nil && currentState.HostCredentials != nil {
		if planState.HostCredentials.Username != currentState.HostCredentials.Username {
			return true
		}
		if planState.HostCredentials.Password != currentState.HostCredentials.Password {
			return true
		}
		if planState.HostCredentials.ApiKey != currentState.HostCredentials.ApiKey {
			return true
		}
	}
	if planState.Orchestrator != nil && currentState.Orchestrator == nil {
		return true
	}
	if planState.Orchestrator == nil && currentState.Orchestrator != nil {
		return true
	}
	if planState.Orchestrator != nil && currentState.Orchestrator != nil {
		if planState.Orchestrator.Host != currentState.Orchestrator.Host {
			return true
		}
		if planState.Orchestrator.Port != currentState.Orchestrator.Port {
			return true
		}
		if planState.Orchestrator.UseAuthentication != nil && currentState.Orchestrator.UseAuthentication == nil {
			return true
		}
		if planState.Orchestrator.UseAuthentication == nil && currentState.Orchestrator.UseAuthentication != nil {
			return true
		}
		if planState.Orchestrator.UseAuthentication != nil && currentState.Orchestrator.UseAuthentication != nil {
			if planState.Orchestrator.UseAuthentication.Username != currentState.Orchestrator.UseAuthentication.Username {
				return true
			}
			if planState.Orchestrator.UseAuthentication.Password != currentState.Orchestrator.UseAuthentication.Password {
				return true
			}
			if planState.Orchestrator.UseAuthentication.ApiKey != currentState.Orchestrator.UseAuthentication.ApiKey {
				return true
			}
		}
	}

	return false
}
