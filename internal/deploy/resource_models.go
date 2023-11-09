package deploy

import (
	"terraform-provider-parallels-desktop/internal/clientmodels"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DeployResourceModel describes the resource data model.
type DeployResourceModel struct {
	SshConnection         *DeployResourceSshConnection `tfsdk:"ssh_connection"`
	CurrentVersion        types.String                 `tfsdk:"current_version"`
	CurrentPackerVersion  types.String                 `tfsdk:"current_packer_version"`
	CurrentVagrantVersion types.String                 `tfsdk:"current_vagrant_version"`
	CurrentGitVersion     types.String                 `tfsdk:"current_git_version"`
	License               types.Object                 `tfsdk:"license"`
	ApiConfig             *ParallelsDesktopApiConfig   `tfsdk:"api_config"`
	Api                   types.Object                 `tfsdk:"api"`
}

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

type ParallelsDesktopApi struct {
	Version  types.String `tfsdk:"version"`
	Protocol types.String `tfsdk:"protocol"`
	Host     types.String `tfsdk:"host"`
	Port     types.String `tfsdk:"port"`
	User     types.String `tfsdk:"user"`
	Password types.String `tfsdk:"password"`
}

func (p *ParallelsDesktopApi) MapObject() basetypes.ObjectValue {
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

type ParallelsDesktopApiConfig struct {
	Port             types.String `tfsdk:"port"`
	InstallVersion   types.String `tfsdk:"install_version"`
	RootPassword     types.String `tfsdk:"root_password"`
	HmacSecret       types.String `tfsdk:"hmac_secret"`
	EncryptionRsaKey types.String `tfsdk:"encryption_rsa_key"`
	LogLevel         types.String `tfsdk:"log_level"`
	EnableTLS        types.Bool   `tfsdk:"enable_tls"`
	TLSPort          types.String `tfsdk:"tls_port"`
	TLSCertificate   types.String `tfsdk:"tls_certificate"`
	TLSPrivateKey    types.String `tfsdk:"tls_private_key"`
}

func (p *ParallelsDesktopApiConfig) MapObject() basetypes.ObjectValue {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["port"] = types.StringType
	attributeTypes["install_version"] = types.StringType
	attributeTypes["root_password"] = types.StringType
	attributeTypes["hmac_secret"] = types.StringType
	attributeTypes["encryption_rsa_key"] = types.StringType
	attributeTypes["log_level"] = types.StringType
	attributeTypes["enable_tls"] = types.BoolType
	attributeTypes["tls_port"] = types.StringType
	attributeTypes["tls_certificate"] = types.StringType
	attributeTypes["tls_private_key"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["api_port"] = p.Port
	attrs["install_version"] = p.InstallVersion
	attrs["root_password"] = p.RootPassword
	attrs["hmac_secret"] = p.HmacSecret
	attrs["encryption_rsa_key"] = p.EncryptionRsaKey
	attrs["log_level"] = p.LogLevel
	attrs["enable_tls"] = p.EnableTLS
	attrs["host_tls_port"] = p.TLSPort
	attrs["tls_certificate"] = p.TLSCertificate
	attrs["tls_private_key"] = p.TLSPrivateKey

	return types.ObjectValueMust(attributeTypes, attrs)
}
