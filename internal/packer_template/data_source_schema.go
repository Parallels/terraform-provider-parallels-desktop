package packertemplate

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/filter"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var packerTemplateDataSourceSchema = schema.Schema{
	MarkdownDescription: "Packer Template Data Source",
	Blocks: map[string]schema.Block{
		authenticator.SchemaName: authenticator.SchemaBlock,
		filter.SchemaName:        filter.SchemaBlock,
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop Api host",
			Description:         "Parallels Desktop Api host",
			Required:            true,
		},
		"templates": schema.ListNestedAttribute{
			MarkdownDescription: "Templates",
			Computed:            true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Id",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Name",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description",
						Computed:            true,
					},
					"internal": schema.BoolAttribute{
						MarkdownDescription: "Internal",
						Computed:            true,
					},
					"specs": schema.MapAttribute{
						MarkdownDescription: "Specs",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"addons": schema.SetAttribute{
						MarkdownDescription: "Addons",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"defaults": schema.MapAttribute{
						MarkdownDescription: "Defaults",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"variables": schema.MapAttribute{
						MarkdownDescription: "Variables",
						Computed:            true,
						ElementType:         types.StringType,
					},
				},
			},
		},
	},
}
