package deploy

import (
	"context"
	"encoding/json"
	"testing"

	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/deploy/schemas"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestReleasedVersionTwoStateUpgrade(t *testing.T) {
	ctx := context.Background()
	// This is the actual previously released schema version 2, including V3-model fields.
	prior := map[string]interface{}{"api_config": map[string]interface{}{"prefix": "/custom", "port": "9090", "root_password": "quotes'\"$`\nsecret", "environment_variables": map[string]string{"CUSTOM": "value"}}, "api": map[string]interface{}{"version": "1.1.0", "host": "mac", "port": "9090", "protocol": "http", "user": "root@localhost", "password": "quotes'\"$`\nsecret"}, "is_registered_in_orchestrator": true, "orchestrator_host_id": "known-id", "orchestrator_host": "http://mac:9090/custom", "keep_after_error": true, "installed_dependencies": []string{"git"}}
	bytes, err := json.Marshal(prior)
	if err != nil {
		t.Fatal(err)
	}
	response := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemas.DeployResourceSchemaV3}}
	upgrader := (&DeployResource{}).UpgradeState(ctx)[2]
	upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{RawState: &tfprotov6.RawState{JSON: bytes}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
	var data models.DeployResourceModelV3
	if d := response.State.Get(ctx, &data); d.HasError() {
		t.Fatal(d)
	}
	if data.ApiConfig.Port.ValueString() != "9090" || data.ApiConfig.RootPassword.ValueString() != prior["api_config"].(map[string]interface{})["root_password"] || !data.ApiConfig.EnabledModules.IsNull() || data.OrchestratorHostId.ValueString() != "known-id" || !data.KeepAfterError.ValueBool() {
		t.Fatal("upgrade lost existing state or invented configured modules")
	}
	if d := response.State.Set(ctx, &data); d.HasError() {
		t.Fatal(d)
	}
	if schemas.DeployResourceSchemaV3.Version != 3 {
		t.Fatal("schema version not bumped")
	}
}

func TestPartialFailureStateCanBePersisted(t *testing.T) {
	ctx := context.Background()
	raw := &tfprotov6.RawState{JSON: []byte(`{"api":{"version":"1.1.0","host":"mac","port":"8080","protocol":"http","user":"root@localhost","password":"secret"},"orchestrator_registration":{"host_id":"confirmed-id","description":"mac"}}`)}
	value, err := raw.Unmarshal(schemas.DeployResourceSchemaV3.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	state := tfsdk.State{Schema: schemas.DeployResourceSchemaV3, Raw: value}
	var data models.DeployResourceModelV3
	if d := state.Get(ctx, &data); d.HasError() {
		t.Fatal(d)
	}
	data.CurrentVersion = types.StringUnknown()
	data.InstalledDependencies = types.ListUnknown(types.StringType)
	data.OrchestratorHostId = types.StringValue("confirmed-id")
	data.OrchestratorHost = types.StringValue("http://mac:8080/api")
	data.IsRegisteredInOrchestrator = types.BoolValue(true)
	preparePartialDeploymentState(ctx, &data, []string{"git"})
	if d := state.Set(ctx, &data); d.HasError() {
		t.Fatal(d)
	}
	var restored models.DeployResourceModelV3
	if d := state.Get(ctx, &restored); d.HasError() {
		t.Fatal(d)
	}
	if restored.OrchestratorHostId.ValueString() != "confirmed-id" || restored.CurrentVersion.IsUnknown() || len(restored.InstalledDependencies.Elements()) != 1 {
		t.Fatal("partial state lost confirmed resources")
	}
}

func TestConfigurationValidationDefersUnknownsAndRejectsConflicts(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		input string
		fail  bool
	}{
		{`{"api_config":{"port":"8080","environment_variables":{"API_PORT":"3080"}}}`, true},
		{`{"api_config":{"devops_version":"main"}}`, true},
		{`{"api_config":{"devops_version":"1.0.3"}}`, true},
		{`{"api_config":{"port":"9090"}}`, false},
	} {
		raw := tfprotov6.RawState{JSON: []byte(tc.input)}
		value, err := raw.Unmarshal(schemas.DeployResourceSchemaV3.Type().TerraformType(ctx))
		if err != nil {
			t.Fatal(err)
		}
		response := resource.ValidateConfigResponse{}
		(&DeployResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemas.DeployResourceSchemaV3, Raw: value}}, &response)
		if response.Diagnostics.HasError() != tc.fail {
			t.Fatalf("unexpected validation result %v", response.Diagnostics)
		}
	}
}

func TestConfigurationValidationDefersAnUnknownBlock(t *testing.T) {
	ctx := context.Background()
	raw := tfprotov6.RawState{JSON: []byte(`{}`)}
	value, err := raw.Unmarshal(schemas.DeployResourceSchemaV3.Type().TerraformType(ctx))
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]tftypes.Value
	if err := value.As(&fields); err != nil {
		t.Fatal(err)
	}
	fields["api_config"] = tftypes.NewValue(fields["api_config"].Type(), tftypes.UnknownValue)
	response := resource.ValidateConfigResponse{}
	(&DeployResource{}).ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemas.DeployResourceSchemaV3, Raw: tftypes.NewValue(value.Type(), fields)}}, &response)
	if response.Diagnostics.HasError() {
		t.Fatal(response.Diagnostics)
	}
}
