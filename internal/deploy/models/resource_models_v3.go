package models

import (
	"strings"

	"terraform-provider-parallels-desktop/internal/apiclient"
	"terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// DeployResourceModel describes the resource data model.

type DeployResourceModelV3 struct {
	SshConnection              *DeployResourceSshConnection           `tfsdk:"ssh_connection"`
	CurrentVersion             types.String                           `tfsdk:"current_version"`
	CurrentPackerVersion       types.String                           `tfsdk:"current_packer_version"`
	CurrentVagrantVersion      types.String                           `tfsdk:"current_vagrant_version"`
	CurrentGitVersion          types.String                           `tfsdk:"current_git_version"`
	ExternalIp                 types.String                           `tfsdk:"external_ip"`
	License                    types.Object                           `tfsdk:"license"`
	Orchestrator               *orchestrator.OrchestratorRegistration `tfsdk:"orchestrator_registration"`
	ReverseProxyHosts          []*reverseproxy.ReverseProxyHost       `tfsdk:"reverse_proxy_host"`
	ApiConfig                  *ParallelsDesktopDevopsConfigV3        `tfsdk:"api_config"`
	Api                        types.Object                           `tfsdk:"api"`
	InstalledDependencies      types.List                             `tfsdk:"installed_dependencies"`
	InstallLocal               types.Bool                             `tfsdk:"install_local"`
	IsRegisteredInOrchestrator types.Bool                             `tfsdk:"is_registered_in_orchestrator"`
	OrchestratorHost           types.String                           `tfsdk:"orchestrator_host"`
	OrchestratorHostId         types.String                           `tfsdk:"orchestrator_host_id"`
}

type ParallelsDesktopDevopsConfigV3 struct {
	Port                                         types.String            `tfsdk:"port" json:"port,omitempty"`
	Prefix                                       types.String            `tfsdk:"prefix" json:"prefix,omitempty"`
	DevOpsVersion                                types.String            `tfsdk:"devops_version" json:"devops_version,omitempty"`
	RootPassword                                 types.String            `tfsdk:"root_password" json:"root_password,omitempty"`
	HmacSecret                                   types.String            `tfsdk:"hmac_secret" json:"hmac_secret,omitempty"`
	EncryptionRsaKey                             types.String            `tfsdk:"encryption_rsa_key" json:"encryption_rsa_key,omitempty"`
	LogLevel                                     types.String            `tfsdk:"log_level" json:"log_level,omitempty"`
	EnableTLS                                    types.Bool              `tfsdk:"enable_tls" json:"enable_tls,omitempty"`
	TLSPort                                      types.String            `tfsdk:"tls_port" json:"tls_port,omitempty"`
	TLSCertificate                               types.String            `tfsdk:"tls_certificate" json:"tls_certificate,omitempty"`
	TLSPrivateKey                                types.String            `tfsdk:"tls_private_key" json:"tls_private_key,omitempty"`
	DisableCatalogCaching                        types.Bool              `tfsdk:"disable_catalog_caching" json:"disable_catalog_caching,omitempty"`
	CatalogCacheKeepFreeDiskSpace                types.Number            `tfsdk:"catalog_cache_keep_free_disk_space_size"`
	CatalogCacheMaxSize                          types.Number            `tfsdk:"catalog_cache_max_size"`
	CatalogCacheAllowCacheAboveKeepFreeDiskSpace types.Bool              `tfsdk:"catalog_cache_allow_cache_above_keep_free_disk_space"`
	DisableCatalogCachingStream                  types.Bool              `tfsdk:"catalog_cache_disable_stream"`
	TokenDurationMinutes                         types.String            `tfsdk:"token_duration_minutes" json:"token_duration_minutes,omitempty"`
	Mode                                         types.String            `tfsdk:"mode" json:"mode,omitempty"`
	UseOrchestratorResources                     types.Bool              `tfsdk:"use_orchestrator_resources"`
	SystemReservedMemory                         types.String            `tfsdk:"system_reserved_memory"`
	SystemReservedCpu                            types.String            `tfsdk:"system_reserved_cpu"`
	SystemReservedDisk                           types.String            `tfsdk:"system_reserved_disk"`
	EnableLogging                                types.Bool              `tfsdk:"enable_logging"`
	LogPath                                      types.String            `tfsdk:"log_path"`
	EnablePortForwarding                         types.Bool              `tfsdk:"enable_port_forwarding"`
	UseLatestBeta                                types.Bool              `tfsdk:"use_latest_beta"`
	EnvironmentVariables                         map[string]types.String `tfsdk:"environment_variables"`
}

func (p *ParallelsDesktopDevopsConfigV3) MapObject() basetypes.ObjectValue {
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
	attributeTypes["catalog_cache_keep_free_disk_space_size"] = types.NumberType
	attributeTypes["catalog_cache_max_size"] = types.NumberType
	attributeTypes["catalog_cache_allow_cache_above_keep_free_disk_space"] = types.BoolType
	attributeTypes["catalog_cache_disable_stream"] = types.BoolType
	attributeTypes["token_duration_minutes"] = types.StringType
	attributeTypes["mode"] = types.StringType
	attributeTypes["use_orchestrator_resources"] = types.BoolType
	attributeTypes["system_reserved_memory"] = types.StringType
	attributeTypes["system_reserved_cpu"] = types.StringType
	attributeTypes["system_reserved_disk"] = types.StringType
	attributeTypes["enable_logging"] = types.BoolType
	attributeTypes["log_path"] = types.StringType
	attributeTypes["enable_port_forwarding"] = types.BoolType
	attributeTypes["use_latest_beta"] = types.BoolType
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
	attrs["catalog_cache_keep_free_disk_space_size"] = p.CatalogCacheKeepFreeDiskSpace
	attrs["catalog_cache_max_size"] = p.CatalogCacheMaxSize
	attrs["catalog_cache_allow_cache_above_keep_free_disk_space"] = p.CatalogCacheAllowCacheAboveKeepFreeDiskSpace
	attrs["catalog_cache_disable_stream"] = p.DisableCatalogCachingStream
	attrs["token_duration_minutes"] = p.TokenDurationMinutes
	attrs["mode"] = p.Mode
	attrs["use_orchestrator_resources"] = p.UseOrchestratorResources
	attrs["system_reserved_memory"] = p.SystemReservedMemory
	attrs["system_reserved_cpu"] = p.SystemReservedCpu
	attrs["system_reserved_disk"] = p.SystemReservedDisk
	attrs["enable_logging"] = p.EnableLogging
	attrs["log_path"] = p.LogPath
	attrs["enable_port_forwarding"] = p.EnablePortForwarding
	attrs["use_latest_beta"] = p.UseLatestBeta

	envVars := make(map[string]attr.Value)
	for k, v := range p.EnvironmentVariables {
		envVars[k] = v
	}
	attrs["environment_variables"] = types.MapValueMust(types.StringType, envVars)

	return types.ObjectValueMust(attributeTypes, attrs)
}

func (o *DeployResourceModelV3) GenerateApiHostConfig(provider *models.ParallelsProviderModel) apiclient.HostConfig {
	if o.Api.IsNull() || o.Api.IsUnknown() {
		return apiclient.HostConfig{}
	}

	hostConfig := apiclient.HostConfig{
		IsOrchestrator: false,
		Host:           strings.ReplaceAll(o.SshConnection.Host.String(), "\"", ""),
		License:        provider.License.ValueString(),
		Authorization: &authenticator.Authentication{
			Username: types.StringValue(strings.ReplaceAll(o.Api.Attributes()["user"].String(), "\"", "")),
			Password: types.StringValue(strings.ReplaceAll(o.Api.Attributes()["password"].String(), "\"", "")),
		},

		DisableTlsValidation: provider.DisableTlsValidation.ValueBool(),
	}

	api_port := strings.ReplaceAll(o.ApiConfig.Port.ValueString(), "\"", "")
	api_schema := "http"

	if o.ApiConfig.EnableTLS.ValueBool() {
		api_schema = "https"
		api_port = strings.ReplaceAll(o.ApiConfig.TLSPort.ValueString(), "\"", "")
	}

	if api_port != "" {
		hostConfig.Host = hostConfig.Host + ":" + api_port
	}

	hostConfig.Host = api_schema + "://" + hostConfig.Host

	return hostConfig
}
