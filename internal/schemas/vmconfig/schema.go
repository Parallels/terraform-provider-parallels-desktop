package vmconfig

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var SchemaName = "config"
var SchemaBlock = schema.SingleNestedBlock{
	MarkdownDescription: "Virtual Machine config block, this is used set some of the most common settings for a VM",
	Attributes: map[string]schema.Attribute{
		"start_headless": schema.BoolAttribute{
			MarkdownDescription: "Set the VM to start headless, this will stop the VM if it is running",
			Optional:            true,
		},
		"enable_rosetta": schema.BoolAttribute{
			MarkdownDescription: "Enable Rosetta on Apple Silicon, this will stop the VM if it is running",
			Optional:            true,
		},
		"pause_idle": schema.BoolAttribute{
			MarkdownDescription: "Pause the VM when the host is idle, this will stop the VM if it is running",
			Optional:            true,
		},
		"auto_start_on_host": schema.BoolAttribute{
			MarkdownDescription: "Start the VM when the host starts, this will stop the VM if it is running",
			Optional:            true,
		},
	},
}
