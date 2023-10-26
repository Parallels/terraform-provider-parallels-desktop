package models

import "github.com/hashicorp/terraform-plugin-framework/types"

type ParallelsProviderModel struct {
	License           types.String `tfsdk:"license"`
	MyAccountUser     types.String `tfsdk:"my_account_user"`
	MyAccountPassword types.String `tfsdk:"my_account_password"`
}
