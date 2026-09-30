package orchestrator

import (
	"net"
	"net/url"
	"strings"
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
	return connectionURL(o.Host.ValueString(), o.Schema.ValueString(), o.Port.ValueString())
}

type OrchestratorDetails struct {
	Schema            basetypes.StringValue         `tfsdk:"schema"`
	Host              basetypes.StringValue         `tfsdk:"host"`
	Port              basetypes.StringValue         `tfsdk:"port"`
	UseAuthentication *authenticator.Authentication `tfsdk:"authentication"`
}

func (o OrchestratorDetails) GetHost() string {
	return connectionURL(o.Host.ValueString(), o.Schema.ValueString(), o.Port.ValueString())
}

func connectionURL(host, scheme, port string) string {
	if scheme == "" {
		scheme = "http"
	}
	if !strings.Contains(host, "://") {
		if net.ParseIP(strings.Trim(host, "[]")) != nil && strings.Contains(host, ":") {
			host = "[" + strings.Trim(host, "[]") + "]"
		}
		host = scheme + "://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return ""
	}
	if port != "" {
		u.Host = net.JoinHostPort(u.Hostname(), port)
	}
	return strings.TrimRight(u.String(), "/")
}
