package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"terraform-provider-parallels-desktop/internal/apiclient"
)

func TestHostConnectionCredentials(t *testing.T) {
	for _, tc := range []struct {
		name, input, apiKey, username, password, wire string
	}{
		{name: "API key", input: "host=test-key@https://catalog.example:443", apiKey: "test-key", wire: "host=test-key@https://catalog.example:443"},
		{name: "legacy API key", input: "host=api_key:test-key@http://catalog.example:3080", apiKey: "test-key", wire: "host=test-key@http://catalog.example:3080"},
		{name: "case insensitive legacy prefix", input: "host=API_KEY:test-key@http://catalog.example:3080", apiKey: "test-key", wire: "host=test-key@http://catalog.example:3080"},
		{name: "without host prefix", input: "test-key@http://catalog.example:3080", apiKey: "test-key", wire: "host=test-key@http://catalog.example:3080"},
		{name: "default scheme", input: "host=test-key@catalog.example", apiKey: "test-key", wire: "host=test-key@https://catalog.example"},
		{name: "username and password", input: "host=user:password@https://catalog.example", username: "user", password: "password", wire: "host=user:password@https://catalog.example"},
		{name: "email username", input: "host=user@example.com:password@https://catalog.example", username: "user@example.com", password: "password", wire: "host=user@example.com:password@https://catalog.example"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := ParseHostConnectionString(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			auth := config.Authorization
			if auth.ApiKey.ValueString() != tc.apiKey || auth.Username.ValueString() != tc.username || auth.Password.ValueString() != tc.password {
				t.Fatalf("unexpected authentication: %+v", auth)
			}
			if got := FormatHostConnectionString(*config); got != tc.wire {
				t.Errorf("remote catalog connection = %q, want %q", got, tc.wire)
			}
		})
	}
}

func TestHostConnectionRejectsMissingCredentialsOrHost(t *testing.T) {
	for _, input := range []string{"", "host=catalog.example", "host=@catalog.example", "host=test-key@", "host=api_key:@catalog.example", "host=test-key@?parameter=value"} {
		t.Run(input, func(t *testing.T) {
			if _, err := ParseHostConnectionString(input); err == nil {
				t.Fatal("accepted incomplete connection")
			}
		})
	}
}

func TestCatalogLookupUsesAPIKey(t *testing.T) {
	for _, credentials := range []string{"test-key", "api_key:test-key"} {
		t.Run(credentials, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/catalog/test/latest/arm64" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					http.NotFound(w, r)
					return
				}
				if r.Header.Get("X-Api-Key") != "test-key" || r.Header.Get("Authorization") != "" {
					t.Error("catalog lookup did not use API key authentication")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"version":"latest","architecture":"arm64"}`))
			}))
			defer server.Close()
			config, err := ParseHostConnectionString("host=" + credentials + "@" + server.URL)
			if err != nil {
				t.Fatal(err)
			}
			manifest, diagnostics := apiclient.GetCatalogManifest(context.Background(), *config, "test", "latest", "arm64")
			if diagnostics.HasError() || manifest == nil {
				t.Fatalf("catalog lookup failed: %v", diagnostics)
			}
			if requests != 1 {
				t.Errorf("requests = %d, want 1 without a token login", requests)
			}
		})
	}
}
