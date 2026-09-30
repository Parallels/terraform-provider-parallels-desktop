package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	deploymodels "terraform-provider-parallels-desktop/internal/deploy/models"
	providermodels "terraform-provider-parallels-desktop/internal/models"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
	"terraform-provider-parallels-desktop/internal/schemas/orchestrator"
)

func TestHostReadinessRetriesStartupAndStopsOnAuthenticationFailure(t *testing.T) {
	for _, status := range []int{503, 401, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls == 1 {
					w.WriteHeader(status)
					return
				}
				if r.URL.Path == "/custom/v1/auth/token" {
					_, _ = w.Write([]byte(`{"token":"test-token"}`))
					return
				}
				if r.URL.Path != "/custom/v1/machines" || r.Header.Get("Authorization") != "Bearer test-token" {
					t.Errorf("unexpected readiness request %s", r.URL.Path)
				}
				_, _ = w.Write([]byte(`[]`))
			}))
			defer server.Close()
			cfg := deploymodels.EffectiveServiceConfig{Endpoint: server.URL + "/custom", Environment: map[string]string{"ROOT_PASSWORD": "password", "ENABLED_MODULES": "api,host"}}
			err := waitForHost(context.Background(), cfg, false)
			if status == 503 {
				if err != nil || calls != 3 {
					t.Fatalf("startup not retried: %d %v", calls, err)
				}
			} else if err == nil || calls != 1 {
				t.Fatalf("terminal response retried: %d %v", calls, err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := pollService(ctx, time.Second, func(context.Context) (bool, error) { return false, nil }); err == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestRegistrationReconciliation(t *testing.T) {
	for _, scenario := range []string{"new", "existing", "lost-post-response", "lookup-error", "readback-auth-error", "username-password", "duplicate-description"} {
		t.Run(scenario, func(t *testing.T) {
			secret := "quote'\"$`\\\nsecret"
			endpoint := "http://mac.local:9090/custom"
			posts, puts, deletes := 0, 0, 0
			var record *apimodels.OrchestratorHost
			if scenario == "existing" {
				record = &apimodels.OrchestratorHost{ID: "existing-id", Host: "http://mac.local:8080/api", Description: "mac", Enabled: true, State: "healthy"}
			}
			if scenario == "duplicate-description" {
				record = &apimodels.OrchestratorHost{ID: "unrelated-id", Host: "http://other-mac:9090/custom", Description: "mac", Enabled: true, State: "healthy"}
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if scenario == "username-password" {
					if r.URL.Path == "/custom/v1/auth/token" {
						var credentials map[string]string
						_ = json.NewDecoder(r.Body).Decode(&credentials)
						if credentials["email"] != "orchestrator-user" || credentials["password"] != "orchestrator-password" {
							t.Error("wrong login credentials")
						}
						_, _ = w.Write([]byte(`{"token":"orchestrator-token"}`))
						return
					}
					if r.Header.Get("Authorization") != "Bearer orchestrator-token" {
						t.Error("missing orchestrator token")
					}
				} else if r.Header.Get("X-Api-Key") != "orchestrator-key" {
					t.Error("wrong orchestrator credentials")
				}
				if !strings.HasPrefix(r.URL.Path, "/custom/v1/orchestrator/hosts") {
					t.Errorf("wrong registration URL %s", r.URL.Path)
				}
				switch r.Method {
				case http.MethodGet:
					if scenario == "lookup-error" {
						w.WriteHeader(403)
						return
					}
					if strings.HasSuffix(r.URL.Path, "/hosts") {
						if record == nil {
							_, _ = w.Write([]byte(`[]`))
						} else {
							_ = json.NewEncoder(w).Encode([]apimodels.OrchestratorHost{*record})
						}
						return
					}
					if scenario == "readback-auth-error" {
						w.WriteHeader(401)
						return
					}
					_ = json.NewEncoder(w).Encode(record)
				case http.MethodPost, http.MethodPut:
					if r.Method == http.MethodPost {
						posts++
					} else {
						puts++
					}
					var payload apimodels.OrchestratorHostRequest
					if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					if payload.Host != endpoint || payload.Authentication.Password != secret || payload.Authentication.Username != "root@localhost" {
						t.Error("endpoint or host credentials changed")
					}
					id := "created-id"
					if record != nil && r.Method == http.MethodPut {
						id = record.ID
					}
					record = &apimodels.OrchestratorHost{ID: id, Host: payload.Host, Description: "mac", Enabled: true, State: "healthy"}
					if scenario == "lost-post-response" {
						w.WriteHeader(503)
						return
					}
					_ = json.NewEncoder(w).Encode(record)
				case http.MethodDelete:
					deletes++
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			data := deploymodels.DeployResourceModelV3{SshConnection: &deploymodels.DeployResourceSshConnection{Host: types.StringValue("mac.local")}, ApiConfig: &deploymodels.ParallelsDesktopDevopsConfigV3{Port: types.StringValue("9090"), Prefix: types.StringValue("/custom"), RootPassword: types.StringValue(secret)}, Orchestrator: &orchestrator.OrchestratorRegistration{Description: types.StringValue("mac"), Orchestrator: &orchestrator.OrchestratorDetails{Host: types.StringValue(server.URL + "/custom"), UseAuthentication: &authenticator.Authentication{ApiKey: types.StringValue("orchestrator-key")}}}}
			if scenario == "existing" {
				data.OrchestratorHostId = types.StringValue("existing-id")
			}
			if scenario == "username-password" {
				data.Orchestrator.Orchestrator.UseAuthentication = &authenticator.Authentication{Username: types.StringValue("orchestrator-user"), Password: types.StringValue("orchestrator-password")}
			}
			resource := &DeployResource{provider: &providermodels.ParallelsProviderModel{}}
			diags := resource.registerWithOrchestrator(context.Background(), &data, &deploymodels.DeployResourceModelV3{})
			failed := scenario == "lookup-error" || scenario == "readback-auth-error"
			if diags.HasError() != failed {
				t.Fatalf("unexpected registration diagnostics: %v", diags)
			}
			if deletes != 0 {
				t.Fatal("registration deleted existing host")
			}
			if scenario == "lookup-error" {
				if posts+puts != 0 {
					t.Fatal("mutation after lookup failure")
				}
				return
			}
			if data.OrchestratorHostId.ValueString() == "" || !data.IsRegisteredInOrchestrator.ValueBool() {
				t.Fatal("lost confirmed identity")
			}
			if scenario == "existing" {
				if puts != 1 || posts != 0 {
					t.Fatal("did not update existing registration")
				}
			} else if posts != 1 {
				t.Fatalf("expected exactly one POST, got %d", posts)
			}
		})
	}
}

func TestEndpointAuthenticationIsolation(t *testing.T) {
	for _, auth := range []*authenticator.Authentication{nil, {Username: types.StringValue("user")}, {ApiKey: types.StringValue("key"), Password: types.StringValue("pass")}} {
		if validateOrchestratorAuth(auth) == nil {
			t.Fatal("accepted incomplete or ambiguous credentials")
		}
	}
	d := orchestrator.OrchestratorDetails{Host: types.StringValue("::1"), Schema: types.StringValue("https"), Port: types.StringValue("8443")}
	u, err := url.Parse(d.GetHost())
	if err != nil || u.Host != "[::1]:8443" {
		t.Fatal(d.GetHost())
	}
}
