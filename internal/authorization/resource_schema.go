package authorization

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

var authorizationResourceSchema = schema.Schema{
	MarkdownDescription: "Parallels Api Authorization Resource, use this to create additional users, api keys, claims and roles in the host Parallel Desktop instance",
	Description:         "Parallels Api Authorization Resource, use this to create additional users, api keys, claims and roles in the host Parallel Desktop instance",

	Blocks: map[string]schema.Block{
		authenticator.SchemaName: authenticator.SchemaBlock,
		ApiKeySchemaBlockName:    ApiKeySchemaBlock,
		UserSchemaBlockName:      UserSchemaBlock,
		ClaimSchemaBlockName:     ClaimSchemaBlock,
		RoleSchemaBlockName:      RoleSchemaBlock,
	},
	Attributes: map[string]schema.Attribute{
		"host": schema.StringAttribute{
			MarkdownDescription: "Parallels Desktop Api host",
			Description:         "Parallels Desktop Api host",
			Required:            true,
		},
	},
}
