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

	orchestratorRequest := apimodels.OrchestratorHostRequest{
		Host:        helpers.GetHostApiBaseUrl(plan.GetHost()),
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
		Host: plan.Orchestrator.GetHost(),
		Authorization: &authenticator.Authentication{
			Username: plan.Orchestrator.UseAuthentication.Username,
			Password: plan.Orchestrator.UseAuthentication.Password,
			ApiKey:   plan.Orchestrator.UseAuthentication.ApiKey,
		},
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

	if data.HostId.ValueString() == "" {
		diag := apiclient.UnregisterWithOrchestrator(context, hostConfig, data.HostId.ValueString())
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
	} else {
		diag := apiclient.UnregisterWithOrchestrator(context, hostConfig, data.Orchestrator.Host.ValueString())
		if diag.HasError() {
			diagnostics.Append(diag...)
			return diagnostics
		}
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
