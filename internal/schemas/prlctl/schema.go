package prlctl

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var SchemaName = "prlctl"
var SchemaBlock = schema.ListNestedBlock{
	MarkdownDescription: "Virtual Machine config block, this is used set some of the most common settings for a VM",
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"operation": schema.StringAttribute{
				MarkdownDescription: "Set the VM to start headless, this will stop the VM if it is running",
				Optional:            true,
			},
			"flags": schema.ListAttribute{
				MarkdownDescription: "Set the VM flags, this will stop the VM if it is running",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"options": schema.ListNestedAttribute{
				MarkdownDescription: "Set the VM options, this will stop the VM if it is running",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"flag": schema.StringAttribute{
							MarkdownDescription: "Set the VM option flag, this will stop the VM if it is running",
							Optional:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Set the VM option value, this will stop the VM if it is running",
							Optional:            true,
						},
					},
				},
			},
		},
	},
}
