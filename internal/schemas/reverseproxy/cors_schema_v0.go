package reverseproxy

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var CorsSchemaBlockV0 = schema.SingleNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps Reverse Proxy Http Route CORS configuration",
	Description:         "Parallels Desktop DevOps Reverse Proxy Http Route CORS configuration",
	Attributes: map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable CORS",
			Description:         "Enable CORS",
			Optional:            true,
		},
		"allowed_origins": schema.ListAttribute{
			MarkdownDescription: "Allowed origins",
			Description:         "Allowed origins",
			Optional:            true,
			ElementType:         types.StringType,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
		},
		"allowed_methods": schema.ListAttribute{
			MarkdownDescription: "Allowed methods",
			Description:         "Allowed methods",
			Optional:            true,
			ElementType:         types.StringType,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
		},
		"allowed_headers": schema.ListAttribute{
			MarkdownDescription: "Allowed headers",
			Description:         "Allowed headers",
			Optional:            true,
			ElementType:         types.StringType,
			Validators: []validator.List{
				listvalidator.SizeAtLeast(1),
			},
		},
	},
}
