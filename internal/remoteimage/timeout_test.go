package remoteimage

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func timeoutPlan(t *testing.T, create *string) tfsdk.Plan {
	t.Helper()
	sch := schema.Schema{Attributes: map[string]schema.Attribute{
		"timeouts": timeouts.Attributes(context.Background(), timeouts.Opts{Create: true}),
	}}
	timeoutType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"create": tftypes.String,
	}}
	var timeoutValue tftypes.Value
	if create == nil {
		timeoutValue = tftypes.NewValue(timeoutType, nil)
	} else {
		timeoutValue = tftypes.NewValue(timeoutType, map[string]tftypes.Value{
			"create": tftypes.NewValue(tftypes.String, *create),
		})
	}
	return tfsdk.Plan{
		Schema: sch,
		Raw: tftypes.NewValue(tftypes.Object{AttributeTypes: map[string]tftypes.Type{"timeouts": timeoutType}},
			map[string]tftypes.Value{"timeouts": timeoutValue}),
	}
}

func TestOperationTimeoutHonorsConfiguredBlock(t *testing.T) {
	configured := "3h"
	got, diags := operationTimeout(context.Background(), timeoutPlan(t, &configured), 60*time.Minute)
	if diags.HasError() || got != 3*time.Hour {
		t.Fatalf("got %v %v, want 3h", got, diags)
	}
}

func TestOperationTimeoutFallsBackToDefault(t *testing.T) {
	got, diags := operationTimeout(context.Background(), timeoutPlan(t, nil), 90*time.Minute)
	if diags.HasError() || got != 90*time.Minute {
		t.Fatalf("got %v %v, want default 90m", got, diags)
	}
}
