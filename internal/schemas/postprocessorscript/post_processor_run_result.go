package postprocessorscript

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PostProcessorScriptRunResult struct {
	ExitCode types.String `tfsdk:"exit_code"`
	Script   types.String `tfsdk:"script"`
	Stdout   types.String `tfsdk:"stdout"`
	Stderr   types.String `tfsdk:"stderr"`
}

func (s *PostProcessorScriptRunResult) Elements(ctx context.Context) []attr.Value {
	attrs := []attr.Value{
		s.ExitCode,
		s.Script,
		s.Stdout,
		s.Stderr,
	}

	return attrs
}

func (s *PostProcessorScriptRunResult) ElementType(ctx context.Context) attr.Type {
	return basetypes.ObjectType{
		AttrTypes: map[string]attr.Type{
			"exit_code": types.StringType,
			"stdout":    types.StringType,
			"stderr":    types.StringType,
			"script":    types.StringType,
		},
	}
}

func (s *PostProcessorScriptRunResult) MapObject(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	attributeTypes := make(map[string]attr.Type)
	attributeTypes["exit_code"] = types.StringType
	attributeTypes["stdout"] = types.StringType
	attributeTypes["stderr"] = types.StringType
	attributeTypes["script"] = types.StringType

	attrs := map[string]attr.Value{}
	attrs["exit_code"] = s.ExitCode
	attrs["stdout"] = s.Stdout
	attrs["stderr"] = s.Stderr
	attrs["script"] = s.Script

	return types.ObjectValue(attributeTypes, attrs)
}
