package orchestrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"terraform-provider-parallels-desktop/internal/apiclient/apimodels"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"
)

func TestLookupUsesIDBeforeEndpointAndNeverDescription(t *testing.T) {
	for _, scenario := range []string{"id-first", "missing-id", "endpoint", "description-only", "forbidden", "missing-auth"} {
		t.Run(scenario, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if scenario == "forbidden" {
					w.WriteHeader(403)
					return
				}
				_ = json.NewEncoder(w).Encode([]apimodels.OrchestratorHost{{ID: "wrong", Host: "http://mac:8080/api", Description: ""}, {ID: "right", Host: "http://other:8080/api", Description: ""}})
			}))
			defer server.Close()
			data := OrchestratorRegistration{Host: types.StringValue("mac"), Schema: types.StringValue("http"), Port: types.StringValue("8080"), Orchestrator: &OrchestratorDetails{Host: types.StringValue(server.URL), UseAuthentication: &authenticator.Authentication{ApiKey: types.StringValue("key")}}}
			switch scenario {
			case "id-first":
				data.HostId = types.StringValue("right")
			case "missing-id":
				data.HostId = types.StringValue("absent")
			case "description-only":
				data.Host = types.StringValue("unrelated")
			case "missing-auth":
				data.Orchestrator.UseAuthentication = nil
			}
			found, record, diagnostics := IsAlreadyRegistered(context.Background(), data, false)
			if scenario == "forbidden" || scenario == "missing-auth" {
				if !diagnostics.HasError() || found || record != nil {
					t.Fatal("lookup error was discarded")
				}
				if scenario == "missing-auth" && calls != 0 {
					t.Fatal("called API without authentication")
				}
				return
			}
			if diagnostics.HasError() {
				t.Fatal(diagnostics)
			}
			switch scenario {
			case "id-first":
				if !found || record.ID != "right" {
					t.Fatal("endpoint took precedence over ID")
				}
			case "endpoint":
				if !found || record.ID != "wrong" {
					t.Fatal("endpoint lookup failed")
				}
			default:
				if found || record != nil {
					t.Fatal("adopted an unrelated host")
				}
			}
		})
	}
}
