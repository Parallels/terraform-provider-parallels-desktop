package deploy

import (
	"terraform-provider-parallels/internal/clientmodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DeployResourceModel describes the resource data model.
type DeployResourceModel struct {
	ApiPort           types.String                 `tfsdk:"api_port"`
	InstallApiVersion types.String                 `tfsdk:"install_api_version"`
	SshConnection     *DeployResourceSshConnection `tfsdk:"ssh_connection"`
	CurrentVersion    types.String                 `tfsdk:"current_version"`
	License           types.Object                 `tfsdk:"license"`
	Api               types.Object                 `tfsdk:"api"`
}

type DeployResourceSshConnection struct {
	Host       types.String `tfsdk:"host"`
	HostPort   types.String `tfsdk:"host_port"`
	User       types.String `tfsdk:"user"`
	Password   types.String `tfsdk:"password"`
	PrivateKey types.String `tfsdk:"private_key"`
}

type ParallelsLicense struct {
	State      types.String `tfsdk:"state"`
	Key        types.String `tfsdk:"key"`
	Restricted types.Bool   `tfsdk:"restricted"`
}

func (p *ParallelsLicense) FromClientModel(value clientmodels.ParallelsServerLicense) {
	p.State = types.StringValue(value.State)
	p.Key = types.StringValue(value.Key)
	p.Restricted = types.BoolValue(value.Restricted == "true")
}

func (p *ParallelsLicense) MapObject() basetypes.ObjectValue {
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

type ParallelsApi struct {
	Version  types.String `tfsdk:"version"`
	Protocol types.String `tfsdk:"protocol"`
	Host     types.String `tfsdk:"host"`
	Port     types.String `tfsdk:"port"`
	User     types.String `tfsdk:"user"`
	Password types.String `tfsdk:"password"`
}

func (p *ParallelsApi) MapObject() basetypes.ObjectValue {
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
