package reverseproxy

import "github.com/hashicorp/terraform-plugin-framework/resource/schema"

var HostBlockV0 = schema.ListNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps Reverse Proxy configuration",
	Description:         "Parallels Desktop DevOps Reverse Proxy configuration",
	NestedObject: schema.NestedBlockObject{
		Blocks: map[string]schema.Block{
			"tcp_route":   TcpRouteSchemaBlockV0,
			"cors":        CorsSchemaBlockV0,
			"tls":         TlsSchemaBlockV0,
			"http_routes": HttpRouteSchemaBlockV0,
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy Host id",
				Description:         "Reverse proxy Host id",
				Computed:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy host",
				Description:         "Reverse proxy host",
				Optional:            true,
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy port",
				Description:         "Reverse proxy port",
				Required:            true,
			},
		},
	},
}
