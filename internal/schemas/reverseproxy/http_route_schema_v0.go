package reverseproxy

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var HttpRouteSchemaBlockV0 = schema.ListNestedBlock{
	MarkdownDescription: "Parallels Desktop DevOps Reverse Proxy Http Route CORS configuration",
	Description:         "Parallels Desktop DevOps Reverse Proxy Http Route CORS configuration",
	NestedObject: schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"path": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route path",
				Description:         "Reverse proxy HTTP Route path",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRelative().AtName("path").AtParent(),
						path.MatchRelative().AtName("pattern").AtParent(),
					}...),
				},
			},
			"target_host": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route target host",
				Description:         "Reverse proxy HTTP Route target host",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRelative().AtName("target_host").AtParent(),
						path.MatchRelative().AtName("target_vm_id").AtParent(),
					}...),
				},
			},
			"target_vm_id": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route target VM id",
				Description:         "Reverse proxy HTTP Route target VM id",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRelative().AtName("target_host").AtParent(),
						path.MatchRelative().AtName("target_vm_id").AtParent(),
					}...),
				},
			},
			"target_port": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route target port",
				Description:         "Reverse proxy HTTP Route target port",
				Optional:            true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route schema",
				Description:         "Reverse proxy HTTP Route schema",
				Optional:            true,
			},
			"pattern": schema.StringAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route pattern",
				Description:         "Reverse proxy HTTP Route pattern",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{
						path.MatchRelative().AtName("path").AtParent(),
						path.MatchRelative().AtName("pattern").AtParent(),
					}...),
				},
			},
			"request_headers": schema.MapAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route request headers",
				Description:         "Reverse proxy HTTP Route request headers",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"response_headers": schema.MapAttribute{
				MarkdownDescription: "Reverse proxy HTTP Route response headers",
				Description:         "Reverse proxy HTTP Route response headers",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	},
}
