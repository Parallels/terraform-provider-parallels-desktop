package virtualmachine

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var virtualMachineDataSourceSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Required: true,
		},
		"filter": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"state": schema.StringAttribute{
					Optional: true,
					Validators: []validator.String{
						stringvalidator.OneOf("running", "suspended", "stopped", "paused", "unknown"),
					},
				},
				"id": schema.StringAttribute{
					Optional: true,
				},
				"name": schema.StringAttribute{
					Optional: true,
				},
			},
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
