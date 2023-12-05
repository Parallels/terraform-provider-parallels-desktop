package postprocessorscript

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PostProcessorScriptRetry struct {
	Attempts            types.Int64  `tfsdk:"attempts"`
	WaitBetweenAttempts types.String `tfsdk:"wait_between_attempts"`
}

func (s *PostProcessorScriptRetry) Elements(ctx context.Context) []attr.Value {
	attrs := []attr.Value{
		s.Attempts,
		s.WaitBetweenAttempts,
	}

	return attrs
}

func (s *PostProcessorScriptRetry) ElementType(ctx context.Context) attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"attempts":              types.Int64Type,
			"wait_between_attempts": types.StringType,
		},
	}
}

func (s *PostProcessorScriptRetry) MapObject(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["attempts"] = types.Int64Type
	attributeTypes["wait_between_attempts"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["attempts"] = s.Attempts
	attrs["wait_between_attempts"] = s.WaitBetweenAttempts

	return types.ObjectValue(attributeTypes, attrs)
}
