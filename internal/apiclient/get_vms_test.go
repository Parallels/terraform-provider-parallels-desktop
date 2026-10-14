package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-parallels-desktop/internal/schemas/authenticator"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The DevOps service's router uses StrictSlash(true), so a path with a trailing slash gets a 301 to the path without it,
// and the provider's HTTP client does not follow redirects. GetVms must use the canonical paths (issue #156).
func TestGetVmsUsesPathsWithoutTrailingSlash(t *testing.T) {
	for _, tc := range []struct {
		orchestrator bool
		path         string
	}{
		{true, "/api/v1/orchestrator/machines"},
		{false, "/api/v1/machines"},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/") {
				http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusMovedPermanently)
				return
			}
			if r.URL.Path != tc.path {
				t.Errorf("wrong URL %s, want %s", r.URL.Path, tc.path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"ID":"1","Name":"vm1"}]`))
		}))
		config := HostConfig{
			Host:           server.URL,
			IsOrchestrator: tc.orchestrator,
			Authorization:  &authenticator.Authentication{ApiKey: types.StringValue("key")},
		}
		vms, diags := GetVms(context.Background(), config, "Name", "vm1")
		server.Close()
		if diags.HasError() {
			t.Fatalf("orchestrator=%v: %v", tc.orchestrator, diags)
		}
		if len(vms) != 1 || vms[0].Name != "vm1" {
			t.Fatalf("orchestrator=%v: got %+v", tc.orchestrator, vms)
		}
	}
}
