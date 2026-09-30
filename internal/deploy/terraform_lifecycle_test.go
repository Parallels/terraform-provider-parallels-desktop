package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	providerschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	providermodels "terraform-provider-parallels-desktop/internal/models"
)

// The fixture provider uses the production resource, schema and protocol server.
// Only OS commands are replaced; HTTP and Terraform plan/apply/state are real.
// This provider exists only in the Go test executable, never in the shipped binary.
type lifecycleProvider struct{}

func (*lifecycleProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bf3"
	resp.Version = "0.0.1"
}
func (*lifecycleProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = providerschema.Schema{}
}
func (*lifecycleProvider) Configure(_ context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	resp.ResourceData = &providermodels.ParallelsProviderModel{License: types.StringValue("fixture-license")}
}
func (*lifecycleProvider) DataSources(context.Context) []func() datasource.DataSource { return nil }
func (*lifecycleProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{func() resource.Resource { return fixtureResource() }}
}

func TestMain(m *testing.M) {
	if os.Getenv("BF3_FIXTURE_PROVIDER") == "1" {
		if err := providerserver.Serve(context.Background(), func() provider.Provider { return &lifecycleProvider{} }, providerserver.ServeOpts{Address: "registry.terraform.io/test/bf3"}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	os.Exit(m.Run())
}

func TestTerraformRegistrationLifecycle(t *testing.T) {
	if os.Getenv("BF3_TERRAFORM_ACCEPTANCE") != "1" {
		t.Skip("set BF3_TERRAFORM_ACCEPTANCE=1 to run the isolated Terraform CLI acceptance test")
	}
	terraform, err := exec.LookPath("terraform")
	if err != nil {
		t.Fatal("Terraform CLI is required for this acceptance test")
	}
	dir := t.TempDir()
	mirror := filepath.Join(dir, "mirror")
	pluginDir := filepath.Join(mirror, "registry.terraform.io", "test", "bf3", "0.0.1", runtime.GOOS+"_"+runtime.GOARCH)
	if err := os.MkdirAll(pluginDir, 0700); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(executable, filepath.Join(pluginDir, "terraform-provider-bf3_v0.0.1")); err != nil {
		t.Fatal(err)
	}
	// Ignore the user's CLI configuration and any dev overrides.
	cliConfig := filepath.Join(dir, "terraform.rc")
	if err := os.WriteFile(cliConfig, []byte("disable_checkpoint = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	fixture := &registrationFixture{}
	server := httptest.NewServer(http.HandlerFunc(fixture.handler))
	defer server.Close()
	endpoint, _ := url.Parse(server.URL)
	run := func(want int, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, terraform, args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "BF3_FIXTURE_PROVIDER=1", "TF_CLI_CONFIG_FILE="+cliConfig, "CHECKPOINT_DISABLE=1", "TF_IN_AUTOMATION=1", "TF_INPUT=0", "TF_LOG=", "TF_REATTACH_PROVIDERS=")
		output, err := command.CombinedOutput()
		code := 0
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				code = exit.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != want {
			t.Fatalf("terraform %v returned %d instead of %d:\n%s", args, code, want, output)
		}
		return output
	}
	write := func(enabled bool, prefix string) {
		t.Helper()
		registration := ""
		if enabled {
			registration = fmt.Sprintf(`
 orchestrator_registration {
   description = "fixture Mac"
   orchestrator {
     host = %q
     authentication { api_key = "fixture-key" }
   }
 }
`, server.URL)
		}
		config := fmt.Sprintf(`terraform {
 required_providers { bf3 = { source = "test/bf3", version = "0.0.1" } }
}
provider "bf3" {}
resource "bf3_deploy" "mac" {
 ssh_connection {
   host = "127.0.0.1"
   user = "fixture"
   password = "fixture"
 }
 api_config {
   port = %q
   prefix = %q
   root_password = "fixture-password"
   devops_version = "1.1.0"
 }
 %s
}
`, endpoint.Port(), prefix, registration)
		if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(config), 0600); err != nil {
			t.Fatal(err)
		}
	}
	values := func() map[string]interface{} {
		t.Helper()
		output := run(0, "show", "-json")
		var state struct {
			Values struct {
				RootModule struct {
					Resources []struct {
						Values map[string]interface{} `json:"values"`
					} `json:"resources"`
				} `json:"root_module"`
			} `json:"values"`
		}
		if err := json.Unmarshal(output, &state); err != nil {
			t.Fatal(err)
		}
		if len(state.Values.RootModule.Resources) != 1 {
			t.Fatal("missing deployment state")
		}
		return state.Values.RootModule.Resources[0].Values
	}
	assertState := func(enabled bool, prefix string) {
		t.Helper()
		v := values()
		if v["is_registered_in_orchestrator"] != enabled {
			t.Fatalf("unexpected registered state %v", v["is_registered_in_orchestrator"])
		}
		if enabled {
			if v["orchestrator_host_id"] != "managed-id" || v["orchestrator_host"] != server.URL+prefix {
				t.Fatal("incorrect confirmed registration")
			}
		} else {
			if v["orchestrator_host_id"] != nil || v["orchestrator_host"] != nil {
				t.Fatal("disabled outputs are not null")
			}
		}
	}
	write(false, "/api")
	run(0, "init", "-backend=false", "-input=false", "-plugin-dir="+mirror)
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(false, "/api")
	run(0, "plan", "-detailed-exitcode", "-input=false")
	fixture.mu.Lock()
	disabledCalls := fixture.calls
	fixture.mu.Unlock()
	if disabledCalls != 0 {
		t.Fatal("disabled create/read/plan contacted orchestrator")
	}
	write(true, "/api")
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(true, "/api")
	run(0, "plan", "-detailed-exitcode", "-input=false")
	write(true, "/custom")
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(true, "/custom")
	// Refresh discovers external removal, and the next apply reconciles it.
	fixture.mu.Lock()
	fixture.record = nil
	fixture.mu.Unlock()
	run(0, "apply", "-refresh-only", "-auto-approve", "-input=false")
	assertState(false, "/custom")
	run(2, "plan", "-detailed-exitcode", "-input=false")
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(true, "/custom")
	// A failed removal retains the old block and ID in Terraform state.
	fixture.mu.Lock()
	fixture.deleteStatus = 403
	fixture.mu.Unlock()
	write(false, "/custom")
	run(1, "apply", "-auto-approve", "-input=false")
	assertState(true, "/custom")
	fixture.mu.Lock()
	fixture.deleteStatus = 0
	fixture.mu.Unlock()
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(false, "/custom")
	run(0, "plan", "-detailed-exitcode", "-input=false")
	// Exercise disabled->disabled Update and registration-enabled destroy.
	fixture.mu.Lock()
	beforeDisabledUpdate := fixture.calls
	fixture.mu.Unlock()
	write(false, "/api")
	run(0, "apply", "-auto-approve", "-input=false")
	assertState(false, "/api")
	fixture.mu.Lock()
	afterDisabledUpdate := fixture.calls
	fixture.mu.Unlock()
	if beforeDisabledUpdate != afterDisabledUpdate {
		t.Fatal("disabled update contacted orchestrator")
	}
	write(true, "/api")
	run(0, "apply", "-auto-approve", "-input=false")
	run(0, "destroy", "-auto-approve", "-input=false")
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if fixture.record != nil {
		t.Fatal("destroy left a managed registration")
	}
	if fixture.posts != 3 {
		t.Fatalf("expected 3 intentional registrations, got %d", fixture.posts)
	}
}
