package models

import (
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DeployResourceModel describes the resource data model.

type DeployResourceModelV0 struct {
	SSHConnection         *DeployResourceSshConnection           `tfsdk:"ssh_connection"`
	CurrentVersion        types.String                           `tfsdk:"current_version"`
	CurrentPackerVersion  types.String                           `tfsdk:"current_packer_version"`
	CurrentVagrantVersion types.String                           `tfsdk:"current_vagrant_version"`
	CurrentGitVersion     types.String                           `tfsdk:"current_git_version"`
	License               types.Object                           `tfsdk:"license"`
	Orchestrator          *orchestrator.OrchestratorRegistration `tfsdk:"orchestrator_registration"`
	APIConfig             *ParallelsDesktopDevopsConfigV0        `tfsdk:"api_config"`
	API                   types.Object                           `tfsdk:"api"`
	InstalledDependencies types.List                             `tfsdk:"installed_dependencies"`
	InstallLocal          types.Bool                             `tfsdk:"install_local"`
}

type ParallelsDesktopDevopsConfigV0 struct {
	Port                     types.String `tfsdk:"port" json:"port,omitempty"`
	Prefix                   types.String `tfsdk:"prefix" json:"prefix,omitempty"`
	DevOpsVersion            types.String `tfsdk:"devops_version" json:"devops_version,omitempty"`
	RootPassword             types.String `tfsdk:"root_password" json:"root_password,omitempty"`
	HmacSecret               types.String `tfsdk:"hmac_secret" json:"hmac_secret,omitempty"`
	EncryptionRsaKey         types.String `tfsdk:"encryption_rsa_key" json:"encryption_rsa_key,omitempty"`
	LogLevel                 types.String `tfsdk:"log_level" json:"log_level,omitempty"`
	EnableTLS                types.Bool   `tfsdk:"enable_tls" json:"enable_tls,omitempty"`
	TLSPort                  types.String `tfsdk:"tls_port" json:"tls_port,omitempty"`
	TLSCertificate           types.String `tfsdk:"tls_certificate" json:"tls_certificate,omitempty"`
	TLSPrivateKey            types.String `tfsdk:"tls_private_key" json:"tls_private_key,omitempty"`
	DisableCatalogCaching    types.Bool   `tfsdk:"disable_catalog_caching" json:"disable_catalog_caching,omitempty"`
	TokenDurationMinutes     types.String `tfsdk:"token_duration_minutes" json:"token_duration_minutes,omitempty"`
	Mode                     types.String `tfsdk:"mode" json:"mode,omitempty"`
	UseOrchestratorResources types.Bool   `tfsdk:"use_orchestrator_resources"`
	SystemReservedMemory     types.String `tfsdk:"system_reserved_memory"`
	SystemReservedCPU        types.String `tfsdk:"system_reserved_cpu"`
	SystemReservedDisk       types.String `tfsdk:"system_reserved_disk"`
	EnableLogging            types.Bool   `tfsdk:"enable_logging"`
}
