package models

import (
	"context"

	"terraform-provider-parallels-desktop/internal/clientmodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type DeployResourceSshConnection struct {
	Host       types.String `tfsdk:"host"`
	HostPort   types.String `tfsdk:"host_port"`
	User       types.String `tfsdk:"user"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
}

type ParallelsDesktopLicense struct {
	State      types.String `tfsdk:"state"`
	Key        types.String `tfsdk:"key"`
	Restricted types.Bool   `tfsdk:"restricted"`
}

func (p *ParallelsDesktopLicense) FromClientModel(value clientmodels.ParallelsServerLicense) {
	p.State = types.StringValue(value.State)
	p.Key = types.StringValue(value.Key)
	p.Restricted = types.BoolValue(value.Restricted == "true")
}

func (p *ParallelsDesktopLicense) MapObject() basetypes.ObjectValue {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["state"] = types.StringType
	attributeTypes["key"] = types.StringType
	attributeTypes["restricted"] = types.BoolType

	attrs := map[string]attr.Value{}
	attrs["state"] = p.State
	attrs["key"] = p.Key
	attrs["restricted"] = p.Restricted

	return types.ObjectValueMust(attributeTypes, attrs)
}

type ParallelsDesktopDevOps struct {
	Version  types.String `tfsdk:"version"`
	Protocol types.String `tfsdk:"protocol"`
	Host     types.String `tfsdk:"host"`
	Port     types.String `tfsdk:"port"`
	User     types.String `tfsdk:"user"`
	Password types.String `tfsdk:"password"`
}

func (p *ParallelsDesktopDevOps) MapObject() basetypes.ObjectValue {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["version"] = types.StringType
	attributeTypes["protocol"] = types.StringType
	attributeTypes["host"] = types.StringType
	attributeTypes["port"] = types.StringType
	attributeTypes["user"] = types.StringType
	attributeTypes["password"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["version"] = p.Version
	attrs["protocol"] = p.Protocol
	attrs["host"] = p.Host
	attrs["port"] = p.Port
	attrs["user"] = p.User
	attrs["password"] = p.Password

	return types.ObjectValueMust(attributeTypes, attrs)
}

func ApiConfigHasChanges(context context.Context, planState, currentState *ParallelsDesktopDevopsConfigV2) bool {
	if planState == nil && currentState == nil {
		return false
	}

	if planState != nil && currentState == nil {
		return true
	}

	if planState == nil && currentState != nil {
		return true
	}

	if planState.Port != currentState.Port {
		return true
	}

	if planState.Prefix != currentState.Prefix {
		return true
	}

	if planState.DevOpsVersion != currentState.DevOpsVersion {
		return true
	}

	if planState.RootPassword != currentState.RootPassword {
		return true
	}

	if planState.HmacSecret != currentState.HmacSecret {
		return true
	}

	if planState.EncryptionRsaKey != currentState.EncryptionRsaKey {
		return true
	}

	if planState.LogLevel != currentState.LogLevel {
		return true
	}

	if planState.EnableTLS != currentState.EnableTLS {
		return true
	}

	if planState.TLSPort != currentState.TLSPort {
		return true
	}

	if planState.TLSCertificate != currentState.TLSCertificate {
		return true
	}

	if planState.TLSPrivateKey != currentState.TLSPrivateKey {
		return true
	}

	if planState.DisableCatalogCaching != currentState.DisableCatalogCaching {
		return true
	}

	if planState.TokenDurationMinutes != currentState.TokenDurationMinutes {
		return true
	}

	if planState.Mode != currentState.Mode {
		return true
	}

	if planState.UseOrchestratorResources != currentState.UseOrchestratorResources {
		return true
	}

	if planState.SystemReservedMemory != currentState.SystemReservedMemory {
		return true
	}

	if planState.SystemReservedCpu != currentState.SystemReservedCpu {
		return true
	}

	if planState.SystemReservedDisk != currentState.SystemReservedDisk {
		return true
	}

	if planState.EnableLogging != currentState.EnableLogging {
		return true
	}

	if len(planState.EnvironmentVariables) != len(currentState.EnvironmentVariables) {
		return true
	}

	if len(planState.EnvironmentVariables) != 0 {
		for k, v := range planState.EnvironmentVariables {
			if currentState.EnvironmentVariables[k] != v {
				return true
			}
		}
	}

	return false
}
