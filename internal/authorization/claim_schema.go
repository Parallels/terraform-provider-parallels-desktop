package authorization

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var ClaimSchemaBlockName = "claim"
var ClaimSchemaBlock = schema.ListNestedBlock{
	MarkdownDescription: "Creates a new claim in the API",
	Description:         "Creates a new claim in the API",
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Claim id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Claim name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	},
}
