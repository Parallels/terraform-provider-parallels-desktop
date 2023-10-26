package deploy

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var deployResourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Parallels Virtual Machine Deployment Resource",
	Attributes: map[string]schema.Attribute{
		"api_port": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop API port",
			Optional:            true,
		},
		"install_api_version": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop API version to install, if empty the latest will be installed",
			Optional:            true,
		},
		"ssh_connection": schema.SingleNestedAttribute{
			MarkdownDescription: "Host connection details",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"host": schema.StringAttribute{
					MarkdownDescription: "Host Machine address",
					Required:            true,
				},
				"host_port": schema.StringAttribute{
					MarkdownDescription: "Host Machine port",
					Optional:            true,
				},
				"user": schema.StringAttribute{
					MarkdownDescription: "Host Machine user",
					Required:            true,
				},
				"password": schema.StringAttribute{
					MarkdownDescription: "Host Machine password",
					Optional:            true,
					Sensitive:           true,
				},
				"private_key": schema.StringAttribute{
					MarkdownDescription: "Host Machine RSA private key",
					Optional:            true,
					Sensitive:           true,
				},
			},
		},
		"current_version": schema.StringAttribute{
			MarkdownDescription: "Current version of Parallels Desktop",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"license": schema.ObjectAttribute{
			MarkdownDescription: "Parallels Desktop license",
			Computed:            true,
			AttributeTypes: map[string]attr.Type{
				"state":      types.StringType,
				"key":        types.StringType,
				"restricted": types.BoolType,
			},
		},
		"api": schema.ObjectAttribute{
			MarkdownDescription: "Parallels Desktop API",
			Computed:            true,
			AttributeTypes: map[string]attr.Type{
				"version":  types.StringType,
				"protocol": types.StringType,
				"host":     types.StringType,
				"port":     types.StringType,
				"user":     types.StringType,
				"password": types.StringType,
			},
		},
	},
}
