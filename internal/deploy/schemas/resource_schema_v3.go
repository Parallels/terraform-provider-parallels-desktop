package schemas

import (
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/schemas/sshconnection"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var DeployResourceSchemaV3 = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Parallels Virtual Machine Deployment Resource",
	Blocks: map[string]schema.Block{
		ApiConfigSchemaName:      ApiConfigSchemaBlockV3,
		reverseproxy.SchemaName:  reverseproxy.HostBlockV0,
		sshconnection.SchemaName: sshconnection.SchemaBlockV0,
		orchestrator.SchemaName:  orchestrator.SchemaBlockV0,
	},
	Version: 2,
	Attributes: map[string]schema.Attribute{
		"current_version": schema.StringAttribute{
			MarkdownDescription: "Current version of Parallels Desktop",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"current_packer_version": schema.StringAttribute{
			MarkdownDescription: "Current version of Hashicorp Packer",
			Description:         "Current version of Hashicorp Packer",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"current_vagrant_version": schema.StringAttribute{
			MarkdownDescription: "Current version of Hashicorp Vagrant",
			Description:         "Current version of Hashicorp Vagrant",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"current_git_version": schema.StringAttribute{
			MarkdownDescription: "Current version of Git",
			Description:         "Current version of Git",
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
		"external_ip": schema.StringAttribute{
			MarkdownDescription: "External IP address",
			Description:         "External IP address",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"api": schema.ObjectAttribute{
			MarkdownDescription: "Parallels Desktop DevOps Service",
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
		"installed_dependencies": schema.ListAttribute{
			MarkdownDescription: "List of installed dependencies",
			Computed:            true,
			ElementType:         types.StringType,
		},
		"install_local": schema.BoolAttribute{
			MarkdownDescription: "Deploy Parallels Desktop in the local machine, this will ignore the need to connect to a remote machine",
			Optional:            true,
		},
		"is_registered_in_orchestrator": schema.BoolAttribute{
			MarkdownDescription: "Is this host registered in the orchestrator",
			Computed:            true,
		},
		"orchestrator_host": schema.StringAttribute{
			MarkdownDescription: "Orchestrator host ID",
			Computed:            true,
		},
		"orchestrator_host_id": schema.StringAttribute{
			MarkdownDescription: "Orchestrator host ID",
			Computed:            true,
		},
	},
}
