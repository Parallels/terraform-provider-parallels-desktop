package sharedfolder

import "github.com/hashicorp/terraform-plugin-framework/resource/schema"

var (
	SchemaName  = "shared_folder"
	SchemaBlock = schema.ListNestedBlock{
		MarkdownDescription: "Shared Folders Block, this is used to share folders with the virtual machine",
		Description:         "Shared Folders Block, this is used to share folders with the virtual machine",
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Shared folder name",
					Optional:            true,
					Description:         "Shared folder name",
				},
				"path": schema.StringAttribute{
					MarkdownDescription: "Path to share",
					Optional:            true,
					Description:         "Path to share",
				},
				"readonly": schema.BoolAttribute{
					MarkdownDescription: "Read only",
					Optional:            true,
					Description:         "Read only",
				},
				"description": schema.StringAttribute{
					MarkdownDescription: "Description",
					Optional:            true,
					Description:         "Description",
				},
				"disabled": schema.BoolAttribute{
					MarkdownDescription: "Disabled",
					Optional:            true,
					Description:         "Disabled",
				},
			},
		},
	}
)
