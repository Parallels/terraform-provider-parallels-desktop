package models

import (
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DeployResourceModel describes the resource data model.

type DeployResourceModelV1 struct {
	SshConnection         *DeployResourceSshConnection           `tfsdk:"ssh_connection"`
	CurrentVersion        types.String                           `tfsdk:"current_version"`
	CurrentPackerVersion  types.String                           `tfsdk:"current_packer_version"`
	CurrentVagrantVersion types.String                           `tfsdk:"current_vagrant_version"`
	CurrentGitVersion     types.String                           `tfsdk:"current_git_version"`
	License               types.Object                           `tfsdk:"license"`
	Orchestrator          *orchestrator.OrchestratorRegistration `tfsdk:"orchestrator_registration"`
	ApiConfig             *ParallelsDesktopDevopsConfigV1        `tfsdk:"api_config"`
	Api                   types.Object                           `tfsdk:"api"`
	InstalledDependencies types.List                             `tfsdk:"installed_dependencies"`
	InstallLocal          types.Bool                             `tfsdk:"install_local"`
}

type ParallelsDesktopDevopsConfigV1 struct {
	Port                     types.String            `tfsdk:"port" json:"port,omitempty"`
	Prefix                   types.String            `tfsdk:"prefix" json:"prefix,omitempty"`
	DevOpsVersion            types.String            `tfsdk:"devops_version" json:"devops_version,omitempty"`
	RootPassword             types.String            `tfsdk:"root_password" json:"root_password,omitempty"`
	HmacSecret               types.String            `tfsdk:"hmac_secret" json:"hmac_secret,omitempty"`
	EncryptionRsaKey         types.String            `tfsdk:"encryption_rsa_key" json:"encryption_rsa_key,omitempty"`
	LogLevel                 types.String            `tfsdk:"log_level" json:"log_level,omitempty"`
	EnableTLS                types.Bool              `tfsdk:"enable_tls" json:"enable_tls,omitempty"`
	TLSPort                  types.String            `tfsdk:"tls_port" json:"tls_port,omitempty"`
	TLSCertificate           types.String            `tfsdk:"tls_certificate" json:"tls_certificate,omitempty"`
	TLSPrivateKey            types.String            `tfsdk:"tls_private_key" json:"tls_private_key,omitempty"`
	DisableCatalogCaching    types.Bool              `tfsdk:"disable_catalog_caching" json:"disable_catalog_caching,omitempty"`
	TokenDurationMinutes     types.String            `tfsdk:"token_duration_minutes" json:"token_duration_minutes,omitempty"`
	Mode                     types.String            `tfsdk:"mode" json:"mode,omitempty"`
	UseOrchestratorResources types.Bool              `tfsdk:"use_orchestrator_resources"`
	SystemReservedMemory     types.String            `tfsdk:"system_reserved_memory"`
	SystemReservedCpu        types.String            `tfsdk:"system_reserved_cpu"`
	SystemReservedDisk       types.String            `tfsdk:"system_reserved_disk"`
	EnableLogging            types.Bool              `tfsdk:"enable_logging"`
	EnvironmentVariables     map[string]types.String `tfsdk:"environment_variables"`
}

func (p *ParallelsDesktopDevopsConfigV1) MapObject() basetypes.ObjectValue {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["port"] = types.StringType
	attributeTypes["devops_version"] = types.StringType
	attributeTypes["root_password"] = types.StringType
	attributeTypes["hmac_secret"] = types.StringType
	attributeTypes["encryption_rsa_key"] = types.StringType
	attributeTypes["log_level"] = types.StringType
	attributeTypes["enable_tls"] = types.BoolType
	attributeTypes["tls_port"] = types.StringType
	attributeTypes["tls_certificate"] = types.StringType
	attributeTypes["tls_private_key"] = types.StringType
	attributeTypes["disable_catalog_caching"] = types.BoolType
	attributeTypes["token_duration_minutes"] = types.StringType
	attributeTypes["mode"] = types.StringType
	attributeTypes["use_orchestrator_resources"] = types.BoolType
	attributeTypes["system_reserved_memory"] = types.StringType
	attributeTypes["system_reserved_cpu"] = types.StringType
	attributeTypes["system_reserved_disk"] = types.StringType
	attributeTypes["enable_logging"] = types.BoolType
	attributeTypes["environment_variables"] = types.MapType{}

	attrs := map[string]attr.Value{}
	attrs["api_port"] = p.Port
	attrs["devops_version"] = p.DevOpsVersion
	attrs["root_password"] = p.RootPassword
	attrs["hmac_secret"] = p.HmacSecret
	attrs["encryption_rsa_key"] = p.EncryptionRsaKey
	attrs["log_level"] = p.LogLevel
	attrs["enable_tls"] = p.EnableTLS
	attrs["host_tls_port"] = p.TLSPort
	attrs["tls_certificate"] = p.TLSCertificate
	attrs["tls_private_key"] = p.TLSPrivateKey
	attrs["disable_catalog_caching"] = p.DisableCatalogCaching
	attrs["token_duration_minutes"] = p.TokenDurationMinutes
	attrs["mode"] = p.Mode
	attrs["use_orchestrator_resources"] = p.UseOrchestratorResources
	attrs["system_reserved_memory"] = p.SystemReservedMemory
	attrs["system_reserved_cpu"] = p.SystemReservedCpu
	attrs["system_reserved_disk"] = p.SystemReservedDisk
	attrs["enable_logging"] = p.EnableLogging

	envVars := make(map[string]attr.Value)
	for k, v := range p.EnvironmentVariables {
		envVars[k] = v
	}
	attrs["environment_variables"] = types.MapValueMust(types.StringType, envVars)

	return types.ObjectValueMust(attributeTypes, attrs)
}
