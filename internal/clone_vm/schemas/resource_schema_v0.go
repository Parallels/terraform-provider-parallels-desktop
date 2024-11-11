package schemas

import (
	"context"

	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/postprocessorscript"
	"terraform-provider-parallels-desktop/internal/schemas/prlctl"
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

func GetCloneVmSchemaV0(ctx context.Context) schema.Schema {
	return schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Parallels Desktop Clone VM resource",
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
			"os_type": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine OS type",
				Computed:            true,
			},
			"base_vm_id": schema.StringAttribute{
				MarkdownDescription: "Base Virtual Machine Id to clone",
				Required:            true,
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
