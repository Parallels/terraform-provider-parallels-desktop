package vmspecs

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SchemaName = "specs"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Virtual Machine Specs block, this is used to set the specs of the virtual machine",
	Description:         "Virtual Machine Specs block, this is used to set the specs of the virtual machine",
	Attributes: map[string]schema.Attribute{
		"force": schema.BoolAttribute{
			MarkdownDescription: "Force the specs to be set, this will stop the VM if it is running",
			Optional:            true,
			Description:         "Force the specs to be set, this will stop the VM if it is running",
		},
		"cpu_count": schema.StringAttribute{
			MarkdownDescription: "The number of CPUs of the virtual machine.",
			Optional:            true,
			Description:         "The number of CPUs of the virtual machine.",
		},
		"memory_size": schema.StringAttribute{
			MarkdownDescription: "The amount of memory of the virtual machine in megabytes.",
			Optional:            true,
			Description:         "The amount of memory of the virtual machine in megabytes.",
		},
		"disk_size": schema.StringAttribute{
			MarkdownDescription: "The size of the disk of the virtual machine in megabytes.",
			Optional:            true,
			Description:         "The size of the disk of the virtual machine in megabytes.",
		},
	},
}
