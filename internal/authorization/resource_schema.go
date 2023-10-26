package authorization

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var authorizationResourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Parallels Api Authorization Resource",

	Blocks: map[string]schema.Block{
		"api_key": schema.ListNestedBlock{
			MarkdownDescription: "Api key",
			PlanModifiers: []planmodifier.List{
				listplanmodifier.RequiresReplace(),
			},
			NestedObject: schema.NestedBlockObject{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Api key id",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Api key name",
						Required:            true,
					},
					"key": schema.StringAttribute{
						MarkdownDescription: "Api key",
						Required:            true,
						Sensitive:           true,
					},
					"secret": schema.StringAttribute{
						MarkdownDescription: "Api key secret",
						Required:            true,
						Sensitive:           true,
					},
					"api_key": schema.StringAttribute{
						MarkdownDescription: "Api key composite",
						Computed:            true,
						Sensitive:           true,
					},
				},
			},
		},
		"user": schema.ListNestedBlock{
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
					},
					"username": schema.StringAttribute{
						MarkdownDescription: "User username",
						Required:            true,
					},
					"email": schema.StringAttribute{
						MarkdownDescription: "User email",
						Required:            true,
					},
					"password": schema.StringAttribute{
						MarkdownDescription: "User password",
						Required:            true,
						Sensitive:           true,
					},
					"roles": schema.ListNestedAttribute{
						MarkdownDescription: "User roles",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "User role id",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "User role name",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("USER", "SUPER_USER"),
									},
								},
							},
						},
					},
					"claims": schema.ListNestedAttribute{
						MarkdownDescription: "User claims",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "User claim id",
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "User claim name",
									Required:            true,
									Validators: []validator.String{
										stringvalidator.OneOf("READ_ONLY",
											"CREATE",
											"DELETE",
											"LIST",
											"CREATE_VM",
											"DELETE_VM",
											"LIST_VM",
											"UPDATE_VM_STATES",
											"UPDATE_VM",
											"CREATE_TEMPLATE",
											"DELETE_TEMPLATE",
											"LIST_TEMPLATE",
											"UPDATE_TEMPLATE"),
									},
								},
							},
						},
					},
				},
			},
		},
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			MarkdownDescription: "Parallels server host",
			Required:            true,
		},
	},
}
