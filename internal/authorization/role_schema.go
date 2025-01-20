package authorization

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var (
	RoleSchemaBlockName = "role"
	RoleSchemaBlock     = schema.ListNestedBlock{
		MarkdownDescription: "Creates a new role in the API",
		Description:         "Creates a new role in the API",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"id": schema.StringAttribute{
					MarkdownDescription: "Role id",
					Computed:            true,
				},
				"name": schema.StringAttribute{
					MarkdownDescription: "Role name",
					Required:            true,
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
			},
		},
	}
)
