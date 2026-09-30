package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	deploymodels "terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/deploy/schemas"
	"terraform-provider-parallels-desktop/internal/interfaces"
	providermodels "terraform-provider-parallels-desktop/internal/models"
)

// All host command execution is replaced. These lifecycle tests never run sudo,
// connect to SSH, install software or change a real launchd job.
type lifecycleCommands struct{}

func (*lifecycleCommands) Username() string { return "fixture" }
func (*lifecycleCommands) Password() string { return "fixture" }
func (*lifecycleCommands) Close() error     { return nil }
func (c *lifecycleCommands) RunCommand(cmd string, args []string) (string, error) {
	return c.RunCommandContext(context.Background(), cmd, args)
}
func (c *lifecycleCommands) RunCommandInput(ctx context.Context, cmd string, args []string, input io.Reader) (string, error) {
	return c.RunCommandContext(ctx, cmd, args)
}
func (*lifecycleCommands) RunCommandContext(ctx context.Context, cmd string, args []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if cmd == "/bin/sh" && len(args) == 4 && args[2] == "discover-tool" {
		return "PRL_TOOL_FOUND:/fixture/" + args[3], nil
	}
	if cmd == "/fixture/prlsrvctl" && len(args) > 0 && args[0] == "info" {
		return `{"version":"27.0.1","license":{"state":"valid","key":"fixture-license","restricted":"false"}}`, nil
	}
	if strings.HasPrefix(cmd, "/fixture/") && len(args) > 0 && args[0] == "--version" {
		return "1.1.0", nil
	}
	if cmd == "sudo" || strings.HasPrefix(cmd, "/Applications/") || strings.HasPrefix(cmd, "/fixture/") || (cmd == "/bin/bash" && strings.Contains(strings.Join(args, " "), "--uninstall")) {
		return "", nil
	}
	return "", fmt.Errorf("unexpected fixture command %s", cmd)
}
func fixtureResource() *DeployResource {
	return &DeployResource{provider: &providermodels.ParallelsProviderModel{License: types.StringValue("fixture-license")}, commandClientFactory: func(context.Context, deploymodels.DeployResourceModelV3) (interfaces.ContextCommandClient, error) {
		return &lifecycleCommands{}, nil
	}}
}

func stateFromJSON(t *testing.T, document map[string]interface{}) tfsdk.State {
	t.Helper()
	bytes, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	raw := tfprotov6.RawState{JSON: bytes}
	value, err := raw.Unmarshal(schemas.DeployResourceSchemaV3.Type().TerraformType(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	return tfsdk.State{Schema: schemas.DeployResourceSchemaV3, Raw: value}
}
func stateModel(t *testing.T, state tfsdk.State) deploymodels.DeployResourceModelV3 {
	t.Helper()
	var data deploymodels.DeployResourceModelV3
	if d := state.Get(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	return data
}
func assertUnregistered(t *testing.T, data deploymodels.DeployResourceModelV3) {
	t.Helper()
	if data.IsRegisteredInOrchestrator.IsUnknown() || data.IsRegisteredInOrchestrator.IsNull() || data.IsRegisteredInOrchestrator.ValueBool() || !data.OrchestratorHost.IsNull() || !data.OrchestratorHostId.IsNull() {
		t.Fatal("expected exactly false/null/null")
	}
	if data.Orchestrator != nil && !data.Orchestrator.HostId.IsNull() {
		t.Fatal("nested host_id was not cleared")
	}
}

type registrationFixture struct {
	mu                          sync.Mutex
	record                      *apimodels.OrchestratorHost
	calls, posts, puts, deletes int
	status, deleteStatus        int
}

func (f *registrationFixture) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if strings.HasSuffix(r.URL.Path, "/auth/token") {
		_, _ = w.Write([]byte(`{"token":"fixture-token"}`))
		return
	}
	if strings.HasSuffix(r.URL.Path, "/machines") {
		_, _ = w.Write([]byte(`[]`))
		return
	}
	f.calls++
	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}
	switch r.Method {
	case http.MethodGet:
		if strings.HasSuffix(r.URL.Path, "/hosts") {
			records := []apimodels.OrchestratorHost{}
			if f.record != nil {
				records = append(records, *f.record)
			}
			_ = json.NewEncoder(w).Encode(records)
			return
		}
		if f.record == nil || !strings.HasSuffix(r.URL.Path, "/"+f.record.ID) {
			w.WriteHeader(404)
			return
		}
		_ = json.NewEncoder(w).Encode(f.record)
	case http.MethodDelete:
		f.deletes++
		if f.deleteStatus != 0 {
			w.WriteHeader(f.deleteStatus)
			return
		}
		if f.record == nil {
			w.WriteHeader(404)
			return
		}
		f.record = nil
		w.WriteHeader(204)
	case http.MethodPost, http.MethodPut:
		if r.Method == http.MethodPost {
			f.posts++
		} else {
			f.puts++
		}
		var request apimodels.OrchestratorHostRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		f.record = &apimodels.OrchestratorHost{ID: "managed-id", Host: request.Host, Description: request.Description, Tags: request.Tags, Enabled: true, State: "healthy"}
		_ = json.NewEncoder(w).Encode(f.record)
	default:
		w.WriteHeader(405)
	}
}
func registrationDocument(server string) map[string]interface{} {
	return map[string]interface{}{"host_id": "nested-id", "orchestrator": map[string]interface{}{"host": server, "authentication": map[string]interface{}{"api_key": "fixture-key"}}}
}

func TestRegistrationReadAndRemovalLifecycle(t *testing.T) {
	ctx := context.Background()
	for _, scenario := range []string{"disabled", "present", "missing", "forbidden", "server-error", "timeout", "delete", "delete-missing", "delete-failed"} {
		t.Run(scenario, func(t *testing.T) {
			f := &registrationFixture{record: &apimodels.OrchestratorHost{ID: "managed-id", Host: "http://mac:8080/api", Enabled: true, State: "healthy"}}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario == "timeout" {
					<-r.Context().Done()
					return
				}
				f.handler(w, r)
			}))
			defer server.Close()
			doc := map[string]interface{}{"install_local": true, "current_version": "27.0.1", "api": map[string]interface{}{"version": "1.1.0", "host": "localhost", "protocol": "http", "port": "8080", "user": "root@localhost", "password": "fixture-license"}, "orchestrator_registration": registrationDocument(server.URL), "is_registered_in_orchestrator": true, "orchestrator_host_id": "managed-id", "orchestrator_host": "http://mac:8080/api"}
			switch scenario {
			case "disabled":
				delete(doc, "orchestrator_registration")
				doc["orchestrator_host_id"] = ""
				doc["orchestrator_host"] = ""
			case "missing", "delete-missing":
				f.record = nil
			case "forbidden":
				f.status = 403
			case "server-error":
				f.status = 503
			case "delete-failed":
				f.deleteStatus = 403
			}
			state := stateFromJSON(t, doc)
			r := fixtureResource()
			if strings.HasPrefix(scenario, "delete") {
				before := stateModel(t, state)
				data := before
				diagnostics := r.unregisterWithOrchestrator(ctx, &data)
				if scenario == "delete-failed" {
					if !diagnostics.HasError() || data.OrchestratorHostId.ValueString() != "managed-id" {
						t.Fatal("failed removal lost identity")
					}
				} else {
					if diagnostics.HasError() {
						t.Fatal(diagnostics)
					}
					assertUnregistered(t, data)
				}
				if f.deletes != 1 {
					t.Fatal("top-level identity was not used for removal")
				}
				return
			}
			resp := resource.ReadResponse{State: state}
			readCtx := ctx
			if scenario == "timeout" {
				var cancel context.CancelFunc
				readCtx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer cancel()
			}
			r.Read(readCtx, resource.ReadRequest{State: state}, &resp)
			data := stateModel(t, resp.State)
			if scenario == "forbidden" || scenario == "server-error" || scenario == "timeout" {
				if !resp.Diagnostics.HasError() || !resp.State.Raw.Equal(state.Raw) {
					t.Fatal("lookup failure changed saved state")
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			if scenario == "disabled" || scenario == "missing" {
				assertUnregistered(t, data)
			} else if data.OrchestratorHostId.ValueString() != "managed-id" || data.Orchestrator.HostId.ValueString() != "managed-id" {
				t.Fatal("read did not refresh authoritative ID")
			}
			if scenario == "disabled" && f.calls != 0 {
				t.Fatal("disabled refresh contacted orchestrator")
			}
			if f.posts+f.puts+f.deletes != 0 {
				t.Fatal("Read mutated a registration")
			}
		})
	}
}

func TestConfirmedRegistrationRejectsInvalidResponses(t *testing.T) {
	for _, record := range []*apimodels.OrchestratorHost{nil, {Host: "http://mac/api"}, {ID: "id"}, {ID: "id", Host: "http://user:secret@mac/api"}} {
		data := deploymodels.DeployResourceModelV3{OrchestratorHostId: types.StringValue("previous")}
		if setRegisteredState(&data, record) == nil || data.OrchestratorHostId.ValueString() != "previous" {
			t.Fatal("invalid response changed identity")
		}
	}
}

func TestConfirmedAbsenceDoesNotAdoptAnExternalRegistration(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`[{"id":"external-id","host":"http://localhost:8080/api"}]`))
	}))
	defer server.Close()
	data := stateModel(t, stateFromJSON(t, map[string]interface{}{"orchestrator_registration": registrationDocument(server.URL)}))
	setUnregisteredState(&data)
	r := fixtureResource()
	if d := r.refreshRegistration(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	if d := r.unregisterWithOrchestrator(context.Background(), &data); d.HasError() {
		t.Fatal(d)
	}
	assertUnregistered(t, data)
	if calls != 0 {
		t.Fatal("confirmed absence led to external registration discovery")
	}
}

func TestEmptySuccessfulRegistrationResponseDoesNotFabricateState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	data := stateModel(t, stateFromJSON(t, map[string]interface{}{"orchestrator_registration": registrationDocument(server.URL)}))
	setUnregisteredState(&data)
	if d := fixtureResource().registerWithOrchestrator(context.Background(), &data, nil); !d.HasError() {
		t.Fatal("empty registration response was accepted")
	}
	assertUnregistered(t, data)
}
