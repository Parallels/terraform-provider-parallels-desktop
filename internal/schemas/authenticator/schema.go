package authenticator

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var SchemaName = "authenticator"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Authenticator block, this is used to authenticate with the Parallels Desktop API, if empty it will try to use the root password",
	Description:         "Authenticator block, this is used to authenticate with the Parallels Desktop API, if empty it will try to use the root password",
	Attributes: map[string]schema.Attribute{
		"username": schema.StringAttribute{
			MarkdownDescription: "Parallels desktop API Username",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.AlsoRequires(path.Expressions{
					path.MatchRelative().AtName("password").AtParent(),
				}...),
			},
		},
		"password": schema.StringAttribute{
			MarkdownDescription: "Parallels desktop API Password",
			Optional:            true,
			Sensitive:           true,
			Validators: []validator.String{
				stringvalidator.AlsoRequires(path.Expressions{
					path.MatchRelative().AtName("username").AtParent(),
				}...),
			},
		},
		"api_key": schema.StringAttribute{
			MarkdownDescription: "Parallels desktop API API Key",
			Optional:            true,
			Sensitive:           true,
		},
	},
}
