package vmspecs

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SchemaName = "specs"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Specs",
	Description:         "Virtual Machine Specs",
	Attributes: map[string]schema.Attribute{
		"force": schema.BoolAttribute{
			MarkdownDescription: "Force the specs to be set, this will stop the VM if it is running",
			Optional:            true,
			Description:         "Force the specs to be set, this will stop the VM if it is running",
		},
		"cpu_count": schema.StringAttribute{
			MarkdownDescription: "Number of CPUs",
			Optional:            true,
			Description:         "The number of CPUs of the virtual machine.",
		},
		"memory_size": schema.StringAttribute{
			MarkdownDescription: "Memory size",
			Optional:            true,
			Description:         "The amount of memory of the virtual machine in megabytes.",
		},
		"disk_size": schema.StringAttribute{
			MarkdownDescription: "Disk size",
			Optional:            true,
			Description:         "The size of the disk of the virtual machine in megabytes.",
		},
	},
}
