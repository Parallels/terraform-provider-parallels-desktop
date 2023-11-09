package authorization

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

var UserSchemaBlockName = "user"
var UserSchemaBlock = schema.ListNestedBlock{
	MarkdownDescription: "User",
	PlanModifiers: []planmodifier.List{
		listplanmodifier.RequiresReplace(),
	},
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "User id",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "User name",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "User username",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				MarkdownDescription: "User email",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "User password",
				Required:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "User roles",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "User role id",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "User role name",
							Required:            true,
						},
					},
				},
			},
			"claims": schema.ListNestedAttribute{
				MarkdownDescription: "User claims",
				Optional:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "User claim id",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "User claim name",
							Required:            true,
						},
					},
				},
			},
		},
	},
}
