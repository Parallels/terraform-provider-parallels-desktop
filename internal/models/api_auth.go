package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type AuthorizationModel struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	ApiKey   types.String `tfsdk:"api_key"`
}
