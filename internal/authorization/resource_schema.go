package authorization

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var authorizationResourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Parallels Api Authorization Resource",

	Blocks: map[string]schema.Block{
		authenticator.SchemaName: authenticator.SchemaBlock,
		ApiKeySchemaBlockName:    ApiKeySchemaBlock,
		UserSchemaBlockName:      UserSchemaBlock,
		ClaimSchemaBlockName:     ClaimSchemaBlock,
		RoleSchemaBlockName:      RoleSchemaBlock,
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			MarkdownDescription: "Parallels server host",
			Required:            true,
		},
	},
}
