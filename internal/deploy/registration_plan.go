package deploy

import (
	"context"
	"terraform-provider-parallels-desktop/internal/deploy/models"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.ResourceWithModifyPlan = &DeployResource{}

func (r *DeployResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var configured types.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("orchestrator_registration"), &configured)...)
	if resp.Diagnostics.HasError() {
		return
	}
	set := func(registered types.Bool, host, id types.String) {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("is_registered_in_orchestrator"), registered)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("orchestrator_host"), host)...)
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("orchestrator_host_id"), id)...)
		if !configured.IsNull() && !configured.IsUnknown() {
			resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("orchestrator_registration").AtName("host_id"), id)...)
		}
	}
	if configured.IsNull() {
		set(types.BoolValue(false), types.StringNull(), types.StringNull())
		return
	}
	set(types.BoolUnknown(), types.StringUnknown(), types.StringUnknown())
	if configured.IsUnknown() || req.State.Raw.IsNull() {
		return
	}
	raw, d := configured.ToTerraformValue(ctx)
	if d != nil || !raw.IsFullyKnown() {
		return
	}
	var previous types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("orchestrator_registration"), &previous)...)
	if previous.IsNull() || previous.IsUnknown() || resp.Diagnostics.HasError() {
		return
	}
	// Only host_id is computed inside the registration block.
	a, b := configured.Attributes(), previous.Attributes()
	for key, value := range a {
		if key != "host_id" && !value.Equal(b[key]) {
			return
		}
	}
	for _, key := range []string{"api_config", "ssh_connection", "install_local"} {
		var desired, prior attr.Value
		switch key {
		case "install_local":
			var x, y types.Bool
			resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(key), &x)...)
			resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(key), &y)...)
			desired, prior = x, y
		default:
			var x, y types.Object
			resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(key), &x)...)
			resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(key), &y)...)
			desired, prior = x, y
		}
		if resp.Diagnostics.HasError() || !desired.Equal(prior) {
			return
		}
		v, err := desired.ToTerraformValue(ctx)
		if err != nil || !v.IsFullyKnown() {
			return
		}
	}
	var registered types.Bool
	var host, id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("is_registered_in_orchestrator"), &registered)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("orchestrator_host"), &host)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("orchestrator_host_id"), &id)...)
	if !resp.Diagnostics.HasError() && registered.ValueBool() && !host.IsUnknown() && host.ValueString() != "" && !id.IsUnknown() && id.ValueString() != "" {
		// Refresh may discover that the remote endpoint drifted while configuration
		// stayed unchanged. Leave outputs unknown so apply restores the desired URL.
		if !req.Config.Raw.IsFullyKnown() {
			return
		}
		var desired models.DeployResourceModelV3
		resp.Diagnostics.Append(req.Config.Get(ctx, &desired)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if r.provider == nil || r.provider.License.IsUnknown() {
			return
		}
		effective, err := deploymentConfig(ctx, &desired, r.provider.License.ValueString())
		if err != nil {
			return
		}
		endpoint, err := canonicalRegisteredHost(effective.Endpoint)
		if err != nil || host.ValueString() != endpoint {
			return
		}
		set(registered, host, id)
	}
}
