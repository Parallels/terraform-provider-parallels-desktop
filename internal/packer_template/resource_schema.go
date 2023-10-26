package packertemplate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func getSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Parallels Virtual Machine State Resource",
		Blocks: map[string]schema.Block{
			"authenticator": schema.SingleNestedBlock{
				MarkdownDescription: "Authenticator",
				Validators: []validator.Object{
					objectvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("username"),
						path.MatchRoot("api_key"),
					}...),
				},
				Attributes: map[string]schema.Attribute{
					"username": schema.StringAttribute{
						MarkdownDescription: "Username",
						Optional:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "Password",
						Optional:            true,
						Sensitive:           true,
						Validators: []validator.String{
							stringvalidator.AlsoRequires(path.Expressions{
								path.MatchRoot("username"),
							}...),
						},
					},
					"api_key": schema.StringAttribute{
						MarkdownDescription: "API Key",
						Optional:            true,
						Sensitive:           true,
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
			"host": schema.StringAttribute{
				MarkdownDescription: "Parallels server host",
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
			"template": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine template",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine name",
				Required:            true,
			},
			"owner": schema.StringAttribute{
				MarkdownDescription: "Virtual Machine owner",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"run_after_create": schema.BoolAttribute{
				MarkdownDescription: "Run after create",
				Optional:            true,
			},
			"specs": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"cpu_count": schema.StringAttribute{
						Optional: true,
					},
					"memory_size": schema.StringAttribute{
						Optional: true,
					},
					"disk_size": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
	}
}
