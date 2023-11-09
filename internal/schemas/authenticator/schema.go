package authenticator

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var SchemaName = "authenticator"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Authenticator",

	Attributes: map[string]schema.Attribute{
		"username": schema.StringAttribute{
			MarkdownDescription: "Username",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.AlsoRequires(path.Expressions{
					path.MatchRoot("password"),
				}...),
			},
		},
		"password": schema.StringAttribute{
			MarkdownDescription: "Password",
			Optional:            true,
			Sensitive:           true,
			Validators: []validator.String{
				stringvalidator.AlsoRequires(path.Expressions{
					path.MatchRoot("username"),
				}...),
			},
		},
		"api_key": schema.StringAttribute{
			MarkdownDescription: "API Key",
			Optional:            true,
			Sensitive:           true,
		},
	},
}
