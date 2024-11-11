package schemas

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/filter"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var VirtualMachineDataSourceSchemaV0 = schema.Schema{
	MarkdownDescription: "Virtual Machine Data Source",
	Blocks: map[string]schema.Block{
		authenticator.SchemaName: authenticator.SchemaBlock,
		filter.SchemaName:        filter.SchemaBlock,
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Required: true,
		},
		"machines": schema.ListNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"host_ip": schema.StringAttribute{
						MarkdownDescription: "The IP address of the host machine",
						Computed:            true,
					},
					"id": schema.StringAttribute{
						MarkdownDescription: "The unique identifier of the virtual machine",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "The name of the virtual machine",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "The description of the virtual machine",
						Computed:            true,
					},
					"os_type": schema.StringAttribute{
						MarkdownDescription: "The type of the operating system installed on the virtual machine",
						Computed:            true,
					},
					"state": schema.StringAttribute{
						MarkdownDescription: "The state of the virtual machine",
						Computed:            true,
					},
					"home": schema.StringAttribute{
						MarkdownDescription: "The path to the virtual machine home directory",
						Computed:            true,
					},
				},
			},
		},
	},
}
