package schemas

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var ApiConfigSchemaBlockV3 = schema.SingleNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps configuration",
	Description:         "Parallels Desktop DevOps configuration",
	Attributes: map[string]schema.Attribute{
		"port": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps port",
			Description:         "Parallels Desktop DevOps port",
			Optional:            true,
		},
		"prefix": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps port",
			Description:         "Parallels Desktop DevOps port",
			Optional:            true,
		},
		"devops_version": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps version to install, if empty the latest will be installed",
			Description:         "Parallels Desktop DevOps version to install, if empty the latest will be installed",
			Optional:            true,
		},
		"use_latest_beta": schema.BoolAttribute{
			MarkdownDescription: "Enables the use of the latest beta",
			Optional:            true,
		},
		"root_password": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps root password",
			Description:         "Parallels Desktop DevOps root password",
			Optional:            true,
			Sensitive:           true,
		},
		"hmac_secret": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps HMAC secret, this is used to sign the JWT tokens",
			Description:         "Parallels Desktop DevOps HMAC secret, this is used to sign the JWT tokens",
			Optional:            true,
			Sensitive:           true,
		},
		"encryption_rsa_key": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps RSA key, this is used to encrypt database file on rest",
			Description:         "Parallels Desktop DevOps RSA key, this is used to encrypt database file on rest",
			Optional:            true,
			Sensitive:           true,
		},
		"log_level": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps log level, you can choose between debug, info, warn, error",
			Description:         "Parallels Desktop DevOps log level, you can choose between debug, info, warn, error",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("debug", "info", "warn", "error"),
			},
		},
		"enable_tls": schema.BoolAttribute{
			MarkdownDescription: "Parallels Desktop DevOps enable TLS",
			Description:         "Parallels Desktop DevOps enable TLS",
			Optional:            true,
		},
		"tls_port": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps TLS port",
			Description:         "Parallels Desktop DevOps TLS port",
			Optional:            true,
		},
		"tls_certificate": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps TLS certificate, this should be a PEM base64 encoded certificate string",
			Description:         "Parallels Desktop DevOps TLS certificate, this should be a PEM base64 encoded certificate string",
			Optional:            true,
			Sensitive:           true,
		},
		"tls_private_key": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop DevOps TLS private key, this should be a PEM base64 encoded private key string",
			Description:         "Parallels Desktop DevOps TLS private key, this should be a PEM base64 encoded private key string",
			Optional:            true,
			Sensitive:           true,
		},
		"disable_catalog_caching": schema.BoolAttribute{
			MarkdownDescription: "Disable catalog caching, this will disable the ability to cache catalog items that are pulled from a remote catalog",
			Description:         "Disable catalog caching, this will disable the ability to cache catalog items that are pulled from a remote catalog",
			Optional:            true,
		},
		"catalog_cache_keep_free_disk_space_size": schema.NumberAttribute{
			MarkdownDescription: "Catalog cache keep free disk space in MB",
			Description:         "Catalog cache keep free disk space in MB",
			Optional:            true,
		},
		"catalog_cache_max_size": schema.NumberAttribute{
			MarkdownDescription: "Catalog cache max size in MB",
			Description:         "Catalog cache max size in MB",
			Optional:            true,
		},
		"catalog_cache_allow_cache_above_keep_free_disk_space": schema.BoolAttribute{
			MarkdownDescription: "Allow catalog cache to override the keep free disk space if it needs, this will render the keep free disk space useless",
			Description:         "Allow catalog cache to override the keep free disk space if it needs, this will render the keep free disk space useless",
			Optional:            true,
		},
		"catalog_cache_disable_stream": schema.BoolAttribute{
			MarkdownDescription: "Disable catalog caching to stream, this will disable the ability of the caching to decompress the catalog items on the fly",
			Description:         "Disable catalog caching to stream, this will disable the ability of the caching to decompress the catalog items on the fly",
			Optional:            true,
		},
		"token_duration_minutes": schema.StringAttribute{
			MarkdownDescription: "JWT Token duration in minutes",
			Description:         "JWT Token duration in minutes",
			Optional:            true,
		},
		"mode": schema.StringAttribute{
			MarkdownDescription: "API Operation mode, either orchestrator or catalog",
			Optional:            true,
			Sensitive:           true,
			Validators: []validator.String{
				stringvalidator.OneOf("orchestrator", "catalog", "api"),
			},
		},
		"use_orchestrator_resources": schema.BoolAttribute{
			MarkdownDescription: "Use orchestrator resources",
			Optional:            true,
		},
		"system_reserved_memory": schema.StringAttribute{
			MarkdownDescription: "System reserved memory in MB",
			Optional:            true,
		},
		"system_reserved_cpu": schema.StringAttribute{
			MarkdownDescription: "System reserved CPU in %",
			Optional:            true,
		},
		"system_reserved_disk": schema.StringAttribute{
			MarkdownDescription: "System reserved disk in MB",
			Optional:            true,
		},
		"enable_logging": schema.BoolAttribute{
			MarkdownDescription: "Enable logging",
			Optional:            true,
		},
		"log_path": schema.StringAttribute{
			MarkdownDescription: "Path to store logs",
			Optional:            true,
		},
		"environment_variables": schema.MapAttribute{
			MarkdownDescription: "Environment variables that can be used in the DevOps service, please see documentation to see which variables are available",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"enable_port_forwarding": schema.BoolAttribute{
			MarkdownDescription: "Enable inbuilt reverse proxy for port forwarding",
			Optional:            true,
		},
	},
}
