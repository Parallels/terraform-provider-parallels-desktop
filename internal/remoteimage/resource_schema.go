package remoteimage

import (
	"context"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"
	"terraform-provider-parallels-desktop/internal/schemas/sharedfolder"
	"terraform-provider-parallels-desktop/internal/schemas/vmconfig"
	"terraform-provider-parallels-desktop/internal/schemas/vmspecs"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func getSchema(ctx context.Context) schema.Schema {
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
		},
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
			"force_changes": schema.BoolAttribute{
				MarkdownDescription: "Force changes, this will force the VM to be stopped and started again",
				Description:         "Force changes, this will force the VM to be stopped and started again",
				Optional:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Parallels Desktop API host",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine Id",
				Computed:            true,
			},
			"os_type": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine OS type",
				Computed:            true,
			},
			"catalog_id": schema.StringAttribute{
				MarkdownDescription: "Catalog Id to pull",
				Description:         "Catalog Id to pull",
				Required:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Catalog version to pull, if empty will pull the 'latest' version",
				Description:         "Catalog version to pull, if empty will pull the 'latest' version",
				Optional:            true,
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
			"host_connection": schema.StringAttribute{
				MarkdownDescription: "Connection",
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
			},
		},
	}
}
