package postprocessorscript

import (
	"context"
	"fmt"

	"terraform-provider-parallels-desktop/internal/apiclient"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PostProcessorScript struct {
	Inline types.List                `tfsdk:"inline"`
	Retry  *PostProcessorScriptRetry `tfsdk:"retry"`
	Result basetypes.ListValue       `tfsdk:"result"`
}

func (s *PostProcessorScript) Apply(ctx context.Context, config apiclient.HostConfig, vmId string) diag.Diagnostics {
	diagnostic := diag.Diagnostics{}
	elements := make([]attr.Value, 0)
	t := PostProcessorScriptRunResult{}

	for _, script := range s.Inline.Elements() {
		value := fmt.Sprintf("%v", script)
		tfValue, err := script.ToTerraformValue(ctx)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("Error converting script to terraform value: %v", err))
			return diagnostic
		}

		var stringScript string
		if err := tfValue.As(&stringScript); err != nil {
			tflog.Error(ctx, fmt.Sprintf("Error converting script to string: %v", err))
			return diagnostic
		}

		resp, resultDiagnostic := apiclient.ExecuteScript(ctx, config, vmId, stringScript)
		if resultDiagnostic.HasError() {
			tflog.Error(ctx, fmt.Sprintf("Error executing script: %v", resultDiagnostic))
			diagnostic = append(diagnostic, resultDiagnostic...)
			return diagnostic
		} else {
			result := PostProcessorScriptRunResult{
				ExitCode: types.StringValue(fmt.Sprintf("%v", resp.ExitCode)),
				Stdout:   types.StringValue(resp.Stdout),
				Stderr:   types.StringValue(resp.Stderr),
				Script:   types.StringValue(value),
			}

			mappedObject, diag := result.MapObject(ctx)
			if diag.HasError() {
				return diag
			}

			elements = append(elements, mappedObject)
			tflog.Info(ctx, fmt.Sprintf("Script %s executed, result %v", value, mappedObject))
		}
	}

	listValue, diag := types.ListValue(t.ElementType(ctx), elements)
	if diag.HasError() {
		return diag
	}

	s.Result = listValue
	return diagnostic
}
