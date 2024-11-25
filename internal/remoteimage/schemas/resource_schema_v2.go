package schemas

import (
	"context"

	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"
	"terraform-provider-parallels-desktop/internal/schemas/reverseproxy"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"
	"terraform-provider-parallels-desktop/internal/schemas/vmconfig"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func GetRemoteImageSchemaV2(ctx context.Context) schema.Schema {
	return schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Parallels Virtual Machine State Resource",
		Blocks: map[string]schema.Block{
			authenticator.SchemaName:       authenticator.SchemaBlock,
			vmspecs.SchemaName:             vmspecs.SchemaBlock,
			postprocessorscript.SchemaName: postprocessorscript.SchemaBlock,
			"on_destroy_script":            postprocessorscript.SchemaBlock,
			sharedfolder.SchemaName:        sharedfolder.SchemaBlock,
			vmconfig.SchemaName:            vmconfig.SchemaBlock,
			prlctl.SchemaName:              prlctl.SchemaBlock,
			reverseproxy.SchemaName:        reverseproxy.HostBlockV0,
		},
		Version: 1,
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
			"force_changes": schema.BoolAttribute{
				MarkdownDescription: "Force changes, this will force the VM to be stopped and started again",
				Optional:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Parallels Desktop DevOps Host",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("orchestrator"),
						path.MatchRoot("host"),
					}...),
				},
			},
			"orchestrator": schema.StringAttribute{
				MarkdownDescription: "Parallels Desktop DevOps Orchestrator",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("orchestrator"),
						path.MatchRoot("host"),
					}...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine Id",
				Computed:            true,
			},
			"orchestrator_host_id": schema.StringAttribute{
				MarkdownDescription: "Orchestrator Host Id if the VM is running in an orchestrator",
				Computed:            true,
			},
			"os_type": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine OS type",
				Computed:            true,
			},
			"catalog_id": schema.StringAttribute{
				MarkdownDescription: "Catalog Id to pull",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Catalog version to pull, if empty will pull the 'latest' version",
				Optional:            true,
			},
			"architecture": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine architecture",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine name to create, this needs to be unique in the host",
				Required:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine owner",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"catalog_connection": schema.StringAttribute{
				MarkdownDescription: "Parallels DevOps Catalog Connection",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				MarkdownDescription: "Path",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"run_after_create": schema.BoolAttribute{
				MarkdownDescription: "Run after create, this will make the VM to run after creation",
				Optional:            true,
				DeprecationMessage:  "Use the `keep_running` attribute instead",
			},
			"external_ip": schema.StringAttribute{
				MarkdownDescription: "VM external IP address",
				Computed:            true,
			},
			"internal_ip": schema.StringAttribute{
				MarkdownDescription: "VM internal IP address",
				Computed:            true,
			},
			"keep_running": schema.BoolAttribute{
				MarkdownDescription: "This will keep the VM running after the terraform apply",
				Optional:            true,
			},
			"host_url": schema.StringAttribute{
				MarkdownDescription: "Parallels Desktop DevOps Host URL",
				Computed:            true,
			},
		},
	}
}
