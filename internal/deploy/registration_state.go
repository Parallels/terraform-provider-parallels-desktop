package deploy

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/helpers"
)

func setUnregisteredState(data *models.DeployResourceModelV3) {
	data.IsRegisteredInOrchestrator = types.BoolValue(false)
	data.OrchestratorHost = types.StringNull()
	data.OrchestratorHostId = types.StringNull()
	if data.Orchestrator != nil {
		data.Orchestrator.HostId = types.StringNull()
	}
}

func canonicalRegisteredHost(host string) (string, error) {
	parsed, err := url.Parse(host)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("orchestrator returned an invalid registered Host API URL")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	return helpers.GetHostApiBaseUrl(parsed.String()), nil
}

func setRegisteredState(data *models.DeployResourceModelV3, record *apimodels.OrchestratorHost) error {
	if record == nil || record.ID == "" {
		return fmt.Errorf("orchestrator returned an empty registration identity")
	}
	host, err := canonicalRegisteredHost(record.Host)
	if err != nil {
		return err
	}
	data.IsRegisteredInOrchestrator = types.BoolValue(true)
	data.OrchestratorHostId = types.StringValue(record.ID)
	data.OrchestratorHost = types.StringValue(host)
	if data.Orchestrator != nil {
		data.Orchestrator.HostId = types.StringValue(record.ID)
	}
	return nil
}

func registrationID(data *models.DeployResourceModelV3) string {
	if id := data.OrchestratorHostId.ValueString(); id != "" {
		return id
	}
	if data.Orchestrator != nil {
		return data.Orchestrator.HostId.ValueString()
	}
	return ""
}

func (r *DeployResource) registrationClient(data *models.DeployResourceModelV3) (apiclient.HostConfig, error) {
	if data.Orchestrator == nil {
		return apiclient.HostConfig{}, fmt.Errorf("registration connection details are absent")
	}
	details := data.Orchestrator.Orchestrator
	if err := validateOrchestratorDetails(details); err != nil {
		return apiclient.HostConfig{}, err
	}
	return apiclient.HostConfig{Host: details.GetHost(), Authorization: details.UseAuthentication, DisableTlsValidation: r.provider.DisableTlsValidation.ValueBool()}, nil
}

// A known ID is authoritative. A 404 must not adopt a different registration
// sharing the same endpoint. Endpoint discovery is only for missing historical IDs.
func (r *DeployResource) lookupRegistration(ctx context.Context, data *models.DeployResourceModelV3) (*apimodels.OrchestratorHost, diag.Diagnostics) {
	var diagnostics diag.Diagnostics
	client, err := r.registrationClient(data)
	if err != nil {
		diagnostics.AddError("Invalid orchestrator configuration", err.Error())
		return nil, diagnostics
	}
	if id := registrationID(data); id != "" {
		record, d := apiclient.GetOrchestratorHost(ctx, client, id)
		if !d.HasError() && record != nil && record.ID != id {
			d.AddError("Invalid registration response", "Returned host ID does not match the managed registration")
			return nil, d
		}
		return record, d
	}
	endpoint := data.OrchestratorHost.ValueString()
	if endpoint == "" {
		cfg, err := deploymentConfig(ctx, data, r.provider.License.ValueString())
		if err != nil {
			diagnostics.AddError("Invalid Host API configuration", err.Error())
			return nil, diagnostics
		}
		endpoint = cfg.Endpoint
	}
	endpoint, err = canonicalRegisteredHost(endpoint)
	if err != nil {
		diagnostics.AddError("Invalid registered host", err.Error())
		return nil, diagnostics
	}
	records, d := apiclient.GetOrchestratorHosts(ctx, client)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return nil, diagnostics
	}
	var found *apimodels.OrchestratorHost
	for i := range records {
		host, e := canonicalRegisteredHost(records[i].Host)
		if e == nil && host == endpoint {
			if found != nil {
				diagnostics.AddError("Ambiguous registration", "Multiple hosts match the managed endpoint; supply or recover the original host ID")
				return nil, diagnostics
			}
			found = &records[i]
		}
	}
	return found, diagnostics
}

// Once absence is confirmed, refresh/removal must not adopt a later external
// registration at the same endpoint. Apply can explicitly reconcile the request.
func registrationConfirmedAbsent(data *models.DeployResourceModelV3) bool {
	return registrationID(data) == "" && !data.IsRegisteredInOrchestrator.IsNull() && !data.IsRegisteredInOrchestrator.IsUnknown() && !data.IsRegisteredInOrchestrator.ValueBool()
}

func (r *DeployResource) refreshRegistration(ctx context.Context, data *models.DeployResourceModelV3) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if data.Orchestrator == nil || registrationConfirmedAbsent(data) {
		setUnregisteredState(data)
		return diagnostics
	}
	record, d := r.lookupRegistration(ctx, data)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return diagnostics
	}
	if record == nil {
		setUnregisteredState(data)
		return diagnostics
	}
	if err := setRegisteredState(data, record); err != nil {
		diagnostics.AddError("Invalid registration response", err.Error())
	}
	return diagnostics
}

func (r *DeployResource) unregisterWithOrchestrator(ctx context.Context, data *models.DeployResourceModelV3) diag.Diagnostics {
	var diagnostics diag.Diagnostics
	if data.Orchestrator == nil || registrationConfirmedAbsent(data) {
		setUnregisteredState(data)
		return diagnostics
	}
	client, err := r.registrationClient(data)
	if err != nil {
		diagnostics.AddError("Invalid orchestrator configuration", err.Error())
		return diagnostics
	}
	id := registrationID(data)
	if id == "" {
		record, d := r.lookupRegistration(ctx, data)
		diagnostics.Append(d...)
		if diagnostics.HasError() {
			return diagnostics
		}
		if record == nil {
			setUnregisteredState(data)
			return diagnostics
		}
		id = record.ID
		if id == "" {
			diagnostics.AddError("Invalid registration response", "Missing host ID")
			return diagnostics
		}
	}
	diagnostics.Append(apiclient.UnregisterWithOrchestrator(ctx, client, id)...)
	if !diagnostics.HasError() {
		setUnregisteredState(data)
	}
	return diagnostics
}
