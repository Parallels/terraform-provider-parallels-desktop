package postprocessorscript

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	SchemaName  = "post_processor_script"
	SchemaBlock = schema.ListNestedBlock{
		MarkdownDescription: "Run any script after the virtual machine is created",
		Description:         "Run any script after the virtual machine is created",
		NestedObject: schema.NestedBlockObject{
			Blocks: map[string]schema.Block{
				"retry": schema.SingleNestedBlock{
					MarkdownDescription: "Retry settings",
					Attributes: map[string]schema.Attribute{
						"attempts": schema.Int64Attribute{
							MarkdownDescription: "Number of attempts",
							Optional:            true,
						},
						"wait_between_attempts": schema.StringAttribute{
							MarkdownDescription: "Wait between attempts, you can use the suffixes 's' for seconds, 'm' for minutes",
							Optional:            true,
						},
					},
				},
			},
			Attributes: map[string]schema.Attribute{
				"inline": schema.ListAttribute{
					MarkdownDescription: "Inline script",
					Optional:            true,
					Description:         "Inline script",
					ElementType:         types.StringType,
				},
				"environment_variables": schema.MapAttribute{
					MarkdownDescription: "Environment variables that can be used in the DevOps service, please see documentation to see which variables are available",
					Optional:            true,
					ElementType:         types.StringType,
				},
				"result": schema.ListNestedAttribute{
					MarkdownDescription: "Result of the script",
					Description:         "Result of the script",
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
)
