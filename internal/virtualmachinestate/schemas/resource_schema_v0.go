package schemas

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var VirtualMachineStateResourceSchemaV0 = schema.Schema{
	MarkdownDescription: "Parallels Virtual Machine State Resource\n Use this to set a virtual machine to a desired state.",
	Blocks: map[string]schema.Block{
		authenticator.SchemaName: authenticator.SchemaBlock,
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			MarkdownDescription: "Parallels server host",
			Required:            true,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Virtual Machine Id",
			Required:            true,
		},
		"operation": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Virtual Machine desired state",
			Validators: []validator.String{
				stringvalidator.OneOf("start", "stop", "suspend", "pause", "resume", "restart"),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"current_state": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Virtual Machine current state",
		},
	},
}
