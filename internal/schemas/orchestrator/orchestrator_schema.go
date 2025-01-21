package orchestrator

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	OrchestratorSchemaName  = "orchestrator"
	OrchestratorSchemaBlock = schema.SingleNestedBlock{
		MarkdownDescription: "Orchestrator connection details",
		Blocks: map[string]schema.Block{
			"authentication": authenticator.SchemaBlock,
		},
		Attributes: map[string]schema.Attribute{
			"schema": schema.StringAttribute{
				MarkdownDescription: "Host Schema",
				Optional:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Host address",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"port": schema.StringAttribute{
				MarkdownDescription: "Host port",
				Optional:            true,
			},
		},
	}
)
