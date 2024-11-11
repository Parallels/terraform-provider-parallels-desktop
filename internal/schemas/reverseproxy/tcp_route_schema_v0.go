package reverseproxy

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var TcpRouteSchemaBlockV0 = schema.SingleNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps Reverse Proxy TCP Route configuration",
	Description:         "Parallels Desktop DevOps Reverse Proxy TCP Route configuration",
	Attributes: map[string]schema.Attribute{
		"target_host": schema.StringAttribute{
			MarkdownDescription: "Reverse proxy host",
			Description:         "Reverse proxy host",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.Expressions{
					path.MatchRelative().AtName("target_host").AtParent(),
					path.MatchRelative().AtName("target_vm_id").AtParent(),
				}...),
			},
		},
		"target_port": schema.StringAttribute{
			MarkdownDescription: "Reverse proxy port",
			Description:         "reverse proxy port",
			Optional:            true,
		},
		"target_vm_id": schema.StringAttribute{
			MarkdownDescription: "Reverse proxy target VM ID",
			Description:         "Reverse proxy target VM ID",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.Expressions{
					path.MatchRelative().AtName("target_host").AtParent(),
					path.MatchRelative().AtName("target_vm_id").AtParent(),
				}...),
			},
		},
	},
}
