package filter

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SchemaName = "filter"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Filter",
	Description:         "Filter",
	Attributes: map[string]schema.Attribute{
		"field_name": schema.StringAttribute{
			MarkdownDescription: "Field name",
			Required:            true,
			Description:         "The field name of the virtual machine where you want to filter.",
		},
		"value": schema.StringAttribute{
			MarkdownDescription: "Value",
			Required:            true,
			Description:         "The value of the virtual machine.",
		},
		"case_insensitive": schema.BoolAttribute{
			MarkdownDescription: "Case insensitive",
			Optional:            true,
		},
	},
}
