package virtualmachine

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/filter"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var virtualMachineDataSourceSchema = schema.Schema{
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
						Computed: true,
					},
					"id": schema.StringAttribute{
						Computed: true,
					},
					"name": schema.StringAttribute{
						Computed: true,
					},
					"description": schema.StringAttribute{
						Computed: true,
					},
					"os_type": schema.StringAttribute{
						Computed: true,
					},
					"state": schema.StringAttribute{
						Computed: true,
					},
					"home": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	},
}
