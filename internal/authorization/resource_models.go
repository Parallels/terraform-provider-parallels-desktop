package authorization

import (
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VirtualMachineStateResourceModel describes the resource data model.
type AuthorizationResourceModel struct {
	Authenticator *authenticator.Authentication             `tfsdk:"authenticator"`
	Host          types.String                              `tfsdk:"host"`
	ApiKeys       []*AuthorizationApiKeysResourceModel      `tfsdk:"api_key"`
	Users         []*AuthorizationUserBlockResourceModel    `tfsdk:"user"`
	Claims        []AuthorizationUserGranularResourceModel  `tfsdk:"claim"`
	Roles         []*AuthorizationUserGranularResourceModel `tfsdk:"role"`
}

type AuthorizationApiKeysResourceModel struct {
	Id     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Key    types.String `tfsdk:"key"`
	Secret types.String `tfsdk:"secret"`
	ApiKey types.String `tfsdk:"api_key"`
}

type AuthorizationUserBlockResourceModel struct {
	Id       types.String                             `tfsdk:"id"`
	Name     types.String                             `tfsdk:"name"`
	Username types.String                             `tfsdk:"username"`
	Email    types.String                             `tfsdk:"email"`
	Password types.String                             `tfsdk:"password"`
	Roles    []AuthorizationUserGranularResourceModel `tfsdk:"roles"`
	Claims   []AuthorizationUserGranularResourceModel `tfsdk:"claims"`
}

type AuthorizationUserGranularResourceModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}
