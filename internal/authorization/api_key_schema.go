package authorization

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var ApiKeySchemaBlockName = "api_key"
var ApiKeySchemaBlock = schema.ListNestedBlock{
	MarkdownDescription: "Create a new Api key in the Parallels desktop api",
	Description:         "Create a new Api key in the Parallels desktop api",
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Api key id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Api key name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Api key",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Api key secret",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Api key composite",
				Computed:            true,
				Sensitive:           true,
			},
		},
	},
}
