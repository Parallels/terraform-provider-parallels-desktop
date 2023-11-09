package filter

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type Filter struct {
	FieldName       types.String `tfsdk:"field_name"`
	Value           types.String `tfsdk:"value"`
	CaseInsensitive types.Bool   `tfsdk:"case_insensitive"`
}

func (s *Filter) Elements(ctx context.Context) []attr.Value {
	attrs := []attr.Value{
		s.FieldName,
		s.Value,
		s.CaseInsensitive,
	}

	return attrs
}

func (s *Filter) ElementType(ctx context.Context) attr.Type {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"field_name":       types.StringType,
			"value":            types.StringType,
			"case_insensitive": types.BoolType,
		},
	}
}

func (s *Filter) MapObject(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["field_name"] = types.StringType
	attributeTypes["value"] = types.StringType
	attributeTypes["case_insensitive"] = types.BoolType

	attrs := map[string]attr.Value{}
	attrs["field_name"] = s.FieldName
	attrs["value"] = s.Value
	attrs["case_insensitive"] = s.CaseInsensitive

	return types.ObjectValue(attributeTypes, attrs)
}
