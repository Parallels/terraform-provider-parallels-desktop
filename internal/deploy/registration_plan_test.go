package deploy

import (
	"context"
	"testing"

	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/deploy/schemas"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestRegistrationPlanTransitions(t *testing.T) {
	ctx := context.Background()
	for _, scenario := range []string{"disabled-create", "disabled-update", "enable", "disable", "unchanged", "api-change", "credentials-change", "missing", "unknown-block", "unknown-auth", "endpoint-drift", "destroy"} {
		t.Run(scenario, func(t *testing.T) {
			configDoc := map[string]interface{}{}
			priorDoc := map[string]interface{}{"is_registered_in_orchestrator": true, "orchestrator_host": "http://localhost:8080/api", "orchestrator_host_id": "managed-id", "orchestrator_registration": registrationDocument("http://orchestrator")}
			priorDoc["orchestrator_registration"].(map[string]interface{})["host_id"] = "managed-id"
			if scenario != "disabled-create" && scenario != "disabled-update" && scenario != "disable" {
				configDoc["orchestrator_registration"] = registrationDocument("http://orchestrator")
				configDoc["orchestrator_registration"].(map[string]interface{})["host_id"] = nil
			}
			if scenario == "disabled-create" || scenario == "enable" || scenario == "disabled-update" {
				priorDoc = map[string]interface{}{}
			}
			if scenario == "endpoint-drift" {
				priorDoc["orchestrator_host"] = "http://other-host:8080/api"
			}
			if scenario == "missing" {
				priorDoc["is_registered_in_orchestrator"] = false
				priorDoc["orchestrator_host"] = nil
				priorDoc["orchestrator_host_id"] = nil
				priorDoc["orchestrator_registration"].(map[string]interface{})["host_id"] = nil
			}
			if scenario == "api-change" {
				configDoc["api_config"] = map[string]interface{}{"port": "9090"}
			}
			if scenario == "credentials-change" {
				configDoc["orchestrator_registration"].(map[string]interface{})["orchestrator"].(map[string]interface{})["authentication"] = map[string]interface{}{"api_key": "different"}
			}
			prior := stateFromJSON(ctx, t, priorDoc)
			configured := stateFromJSON(ctx, t, configDoc)
			if scenario == "disabled-create" || scenario == "enable" {
				prior.Raw = tftypes.NewValue(prior.Raw.Type(), nil)
			}
			if scenario == "unknown-block" || scenario == "unknown-auth" {
				var root map[string]tftypes.Value
				_ = configured.Raw.As(&root)
				if scenario == "unknown-block" {
					root["orchestrator_registration"] = tftypes.NewValue(root["orchestrator_registration"].Type(), tftypes.UnknownValue)
				} else {
					var block, details map[string]tftypes.Value
					_ = root["orchestrator_registration"].As(&block)
					_ = block["orchestrator"].As(&details)
					details["authentication"] = tftypes.NewValue(details["authentication"].Type(), tftypes.UnknownValue)
					block["orchestrator"] = tftypes.NewValue(block["orchestrator"].Type(), details)
					root["orchestrator_registration"] = tftypes.NewValue(root["orchestrator_registration"].Type(), block)
				}
				configured.Raw = tftypes.NewValue(configured.Raw.Type(), root)
			}
			plan := tfsdk.Plan{Schema: schemas.DeployResourceSchemaV3, Raw: configured.Raw}
			if scenario == "destroy" {
				plan.Raw = tftypes.NewValue(plan.Raw.Type(), nil)
			}
			response := resource.ModifyPlanResponse{Plan: plan}
			fixtureResource().ModifyPlan(ctx, resource.ModifyPlanRequest{Config: tfsdk.Config{Schema: schemas.DeployResourceSchemaV3, Raw: configured.Raw}, State: prior, Plan: plan}, &response)
			if response.Diagnostics.HasError() {
				t.Fatal(response.Diagnostics)
			}
			if scenario == "destroy" {
				if !response.Plan.Raw.IsNull() {
					t.Fatal("modified destroy plan")
				}
				return
			}
			var registered types.Bool
			var host, id types.String
			_ = response.Plan.GetAttribute(ctx, path.Root("is_registered_in_orchestrator"), &registered)
			_ = response.Plan.GetAttribute(ctx, path.Root("orchestrator_host"), &host)
			_ = response.Plan.GetAttribute(ctx, path.Root("orchestrator_host_id"), &id)
			switch scenario {
			case "disabled-create", "disabled-update", "disable":
				assertUnregistered(t, models.DeployResourceModelV3{IsRegisteredInOrchestrator: registered, OrchestratorHost: host, OrchestratorHostId: id})
			case "unchanged":
				if !registered.ValueBool() || id.ValueString() != "managed-id" || host.ValueString() == "" {
					t.Fatal("unchanged registration not preserved")
				}
			default:
				if !registered.IsUnknown() || !host.IsUnknown() || !id.IsUnknown() {
					t.Fatal("unconfirmed plan values must remain unknown")
				}
			}
		})
	}
}
