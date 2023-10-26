package virtualmachinestate

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var virtualMachineStateResourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Parallels Virtual Machine State Resource",

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
