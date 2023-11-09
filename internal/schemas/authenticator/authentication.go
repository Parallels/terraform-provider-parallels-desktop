package authenticator

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Authentication struct {
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	ApiKey   types.String `tfsdk:"api_key"`
}

func (s *Authentication) Elements(ctx context.Context) []attr.Value {
	attrs := []attr.Value{
		s.Username,
		s.Password,
		s.ApiKey,
	}

	return attrs
}

func (s *Authentication) ElementType(ctx context.Context) attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"username": types.StringType,
			"password": types.StringType,
			"api_key":  types.StringType,
		},
	}
}

func (s *Authentication) MapObject(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["username"] = types.StringType
	attributeTypes["password"] = types.StringType
	attributeTypes["api_key"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["username"] = s.Username
	attrs["password"] = s.Password
	attrs["api_key"] = s.ApiKey

	return types.ObjectValue(attributeTypes, attrs)
}
