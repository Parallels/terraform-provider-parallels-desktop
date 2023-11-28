package orchestrator

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var SchemaName = "orchestrator_registration"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Orchestrator connection details",
	Blocks: map[string]schema.Block{
		"host_credentials":     authenticator.SchemaBlock,
		OrchestratorSchemaName: OrchestratorSchemaBlock,
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
		"tags": schema.ListAttribute{
			MarkdownDescription: "Host tags",
			Optional:            true,
			ElementType:         types.StringType,
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "Host description",
			Optional:            true,
		},
		"host_id": schema.StringAttribute{
			MarkdownDescription: "Host Orchestrator ID",
			Computed:            true,
		},
	},
}
