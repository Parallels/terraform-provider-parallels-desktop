package sshconnection

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	SchemaName    = "ssh_connection"
	SchemaBlockV0 = schema.SingleNestedBlock{
		MarkdownDescription: "Host connection details",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				MarkdownDescription: "Host Machine address",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"host_port": schema.StringAttribute{
				MarkdownDescription: "Host Machine port",
				Optional:            true,
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "Host Machine user",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Host Machine password",
				Optional:            true,
				Sensitive:           true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Host Machine RSA private key",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
)
