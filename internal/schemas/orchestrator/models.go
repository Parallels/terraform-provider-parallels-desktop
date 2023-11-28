package orchestrator

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type OrchestratorRegistration struct {
	HostId          basetypes.StringValue         `tfsdk:"host_id"`
	Schema          basetypes.StringValue         `tfsdk:"schema"`
	Host            basetypes.StringValue         `tfsdk:"host"`
	Port            basetypes.StringValue         `tfsdk:"port"`
	Description     basetypes.StringValue         `tfsdk:"description"`
	Tags            []string                      `tfsdk:"tags"`
	HostCredentials *authenticator.Authentication `tfsdk:"host_credentials"`
	Orchestrator    *OrchestratorDetails          `tfsdk:"orchestrator"`
}

func (o OrchestratorRegistration) GetHost() string {
	host := o.Host.ValueString()
	if o.Schema.ValueString() != "" {
		host = o.Schema.ValueString() + "://" + host
	}
	if o.Port.ValueString() != "" {
		host = host + ":" + o.Port.ValueString()
	}

	return host
}

type OrchestratorDetails struct {
	Schema            basetypes.StringValue         `tfsdk:"schema"`
	Host              basetypes.StringValue         `tfsdk:"host"`
	Port              basetypes.StringValue         `tfsdk:"port"`
	UseAuthentication *authenticator.Authentication `tfsdk:"authentication"`
}

func (o OrchestratorDetails) GetHost() string {
	host := o.Host.ValueString()
	if o.Schema.ValueString() != "" {
		host = o.Schema.ValueString() + "://" + host
	}
	if o.Port.ValueString() != "" {
		host = host + ":" + o.Port.ValueString()
	}

	return host
}
