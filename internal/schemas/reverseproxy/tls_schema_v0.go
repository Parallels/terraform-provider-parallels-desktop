package reverseproxy

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var TlsSchemaBlockV0 = schema.SingleNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps Reverse Proxy Http Route TLS configuration",
	Description:         "Parallels Desktop DevOps Reverse Proxy Http Route TLS configuration",
	Attributes: map[string]schema.Attribute{
		"enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable TLS",
			Description:         "Enable TLS",
			Optional:            true,
		},
		"certificate": schema.StringAttribute{
			MarkdownDescription: "TLS Certificate",
			Description:         "TLS Certificate",
			Optional:            true,
		},
		"private_key": schema.StringAttribute{
			MarkdownDescription: "TLS Private Key",
			Description:         "TLS Private Key",
			Optional:            true,
		},
	},
}
