package filter

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SchemaName = "filter"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Filter block, this is used to filter data sources",
	Description:         "Filter block, this is used to filter data sources",
	Attributes: map[string]schema.Attribute{
		"field_name": schema.StringAttribute{
			MarkdownDescription: "The field name of the datasource where you want to filter.",
			Required:            true,
			Description:         "The field name of the datasource where you want to filter.",
		},
		"value": schema.StringAttribute{
			MarkdownDescription: "The value you want to filter, this can be a regular expression.",
			Required:            true,
			Description:         "The value you want to filter, this can be a regular expression.",
		},
		"case_insensitive": schema.BoolAttribute{
			MarkdownDescription: "Case insensitive, if true the filter will be case insensitive.",
			Description:         "Case insensitive, if true the filter will be case insensitive.",
			Optional:            true,
		},
	},
}
