package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-parallels-desktop/internal/constants"
	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetVmsCanonicalListPath(t *testing.T) {
	for _, mode := range []struct {
		name           string
		isOrchestrator bool
		path           string
	}{
		{name: "host", path: "/api/v1/machines"},
		{name: "orchestrator", isOrchestrator: true, path: "/api/v1/orchestrator/machines"},
	} {
		for _, filtered := range []bool{false, true} {
			name := mode.name + "/unfiltered"
			if filtered {
				name = mode.name + "/filtered"
			}
			t.Run(name, func(t *testing.T) {
				requests := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests++
					// Reproduce the service's StrictSlash redirect for list paths.
					if strings.HasSuffix(r.URL.Path, "/") {
						http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusMovedPermanently)
						return
					}
					if r.Method != http.MethodGet || r.URL.Path != mode.path {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						http.NotFound(w, r)
						return
					}
					if got := r.Header.Get("X-Api-Key"); got != "test-api-key" {
						t.Errorf("API key = %q, want test-api-key", got)
					}
					wantFilter := ""
					if filtered {
						wantFilter = "Name=test-vm"
					}
					if got := r.Header.Get(constants.FILTER_HEADER); got != wantFilter {
						t.Errorf("filter = %q, want %q", got, wantFilter)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					_, _ = w.Write([]byte(`[{"ID":"vm-id","Name":"test-vm"}]`))
				}))
				defer server.Close()

				config := HostConfig{
					Host:           server.URL + "/",
					IsOrchestrator: mode.isOrchestrator,
					Authorization: &authenticator.Authentication{
						ApiKey: types.StringValue("test-api-key"),
					},
				}
				filterField, filterValue := "", ""
				if filtered {
					filterField, filterValue = "Name", `"test-vm"`
				}
				vms, diagnostics := GetVms(context.Background(), config, filterField, filterValue)
				if diagnostics.HasError() {
					t.Fatalf("GetVms returned errors: %v", diagnostics)
				}
				if len(vms) != 1 || vms[0].ID != "vm-id" || vms[0].Name != "test-vm" {
					t.Fatalf("unexpected VMs: %+v", vms)
				}
				if requests != 1 {
					t.Errorf("requests = %d, want 1", requests)
				}
			})
		}
	}
}
