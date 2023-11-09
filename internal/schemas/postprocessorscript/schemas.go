package postprocessorscript

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var SchemaName = "post_processor_script"
var SchemaBlock = schema.ListNestedBlock{
	MarkdownDescription: "Run any script after the virtual machine is created",
	Description:         "Run any script after the virtual machine is created",
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"inline": schema.ListAttribute{
				MarkdownDescription: "Inline script",
				Optional:            true,
				Description:         "Inline script",
				ElementType:         types.StringType,
			},
			"result": schema.ListNestedAttribute{
				MarkdownDescription: "Inline script",
				Description:         "Inline script",
				Computed:            true,
				Sensitive:           true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"exit_code": schema.StringAttribute{
							MarkdownDescription: "Exit code",
							Optional:            true,
							Description:         "Exit code",
						},
						"stdout": schema.StringAttribute{
							MarkdownDescription: "Stdout",
							Optional:            true,
							Description:         "Stdout",
						},
						"stderr": schema.StringAttribute{
							MarkdownDescription: "Stderr",
							Optional:            true,
							Description:         "Stderr",
						},
						"script": schema.StringAttribute{
							MarkdownDescription: "Script",
							Optional:            true,
							Description:         "Script",
						},
					},
				},
			},
		},
	},
}
