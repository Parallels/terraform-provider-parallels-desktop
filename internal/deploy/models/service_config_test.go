package models

import (
	"context"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveServiceConfiguration(t *testing.T) {
	tests := []struct {
		name                   string
		config                 *ParallelsDesktopDevopsConfigV3
		registration           bool
		endpoint, modules, err string
	}{
		{name: "defaults", endpoint: "http://[::1]:8080/api", modules: "api,host"},
		{name: "env only", config: &ParallelsDesktopDevopsConfigV3{EnvironmentVariables: map[string]types.String{"API_PORT": types.StringValue("9090"), "API_PREFIX": types.StringValue("/custom/")}}, endpoint: "http://[::1]:9090/custom", modules: "api,host"},
		{name: "typed port", config: &ParallelsDesktopDevopsConfigV3{Port: types.StringValue("8081")}, endpoint: "http://[::1]:8081/api", modules: "api,host"},
		{name: "equivalent aliases", config: &ParallelsDesktopDevopsConfigV3{Port: types.StringValue("8080"), EnvironmentVariables: map[string]types.String{"API_PORT": types.StringValue("08080")}}, endpoint: "http://[::1]:8080/api", modules: "api,host"},
		{name: "conflict", config: &ParallelsDesktopDevopsConfigV3{Port: types.StringValue("8080"), EnvironmentVariables: map[string]types.String{"API_PORT": types.StringValue("3080")}}, err: "conflicts"},
		{name: "legacy modules", config: &ParallelsDesktopDevopsConfigV3{Mode: types.StringValue("orchestrator")}, endpoint: "http://[::1]:8080/api", modules: "api,host,orchestrator"},
		{name: "module conflict", config: &ParallelsDesktopDevopsConfigV3{Mode: types.StringValue("api"), EnabledModules: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("api")})}, err: "must agree"},
		{name: "registration host module", config: &ParallelsDesktopDevopsConfigV3{EnvironmentVariables: map[string]types.String{"ENABLED_MODULES": types.StringValue("api")}}, registration: true, err: "host module"},
		{name: "unknown apply", config: &ParallelsDesktopDevopsConfigV3{Port: types.StringUnknown()}, err: "must be known"},
		{name: "unknown env", config: &ParallelsDesktopDevopsConfigV3{EnvironmentVariables: map[string]types.String{"CUSTOM": types.StringUnknown()}}, err: "known string"},
		{name: "invalid port", config: &ParallelsDesktopDevopsConfigV3{Port: types.StringValue("65536")}, err: "65535"},
		{name: "invalid prefix", config: &ParallelsDesktopDevopsConfigV3{Prefix: types.StringValue("/api?token=secret")}, err: "absolute URL path"},
		{name: "version conflict", config: &ParallelsDesktopDevopsConfigV3{DevOpsVersion: types.StringValue("1.1.0"), UseLatestBeta: types.BoolValue(true)}, err: "cannot be combined"},
		{name: "TLS", config: &ParallelsDesktopDevopsConfigV3{EnableTLS: types.BoolValue(true), TLSPort: types.StringValue("9443"), TLSCertificate: types.StringValue("cert"), TLSPrivateKey: types.StringValue("key")}, endpoint: "https://[::1]:9443/api", modules: "api,host"},
		{name: "missing TLS key", config: &ParallelsDesktopDevopsConfigV3{EnableTLS: types.BoolValue(true)}, err: "TLS requires"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before ParallelsDesktopDevopsConfigV3
			if tt.config != nil {
				before = *tt.config
			}
			got, err := ResolveServiceConfig(context.Background(), tt.config, "fallback", "::1", tt.registration)
			if tt.err != "" {
				if err == nil || !strings.Contains(err.Error(), tt.err) {
					t.Fatalf("expected %s, got %v", tt.err, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Endpoint != tt.endpoint || got.Environment["ENABLED_MODULES"] != tt.modules {
				t.Fatalf("unexpected endpoint %s, modules %s", got.Endpoint, got.Environment["ENABLED_MODULES"])
			}
			if tt.config != nil && !reflect.DeepEqual(before, *tt.config) {
				t.Fatal("mutated configured attributes")
			}
		})
	}
}

func TestSensitiveValuesAndFalseZeroMappings(t *testing.T) {
	secret := "quote'\"$`\\\npassword"
	number, _, _ := big.ParseFloat("12345678901234567890.125", 10, 256, big.ToNearestEven)
	cfg := &ParallelsDesktopDevopsConfigV3{RootPassword: types.StringValue(secret), EnableLogging: types.BoolValue(false), EnablePortForwarding: types.BoolValue(false), CatalogCacheAllowCacheAboveKeepFreeDiskSpace: types.BoolValue(false), CatalogCacheKeepFreeDiskSpace: types.NumberValue(big.NewFloat(0)), CatalogCacheMaxSize: types.NumberValue(number), EnvironmentVariables: map[string]types.String{"CUSTOM": types.StringValue("0")}}
	got, err := ResolveServiceConfig(context.Background(), cfg, "fallback", "mac.local", true)
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"ROOT_PASSWORD": secret, "LOG_TO_FILE": "false", "DISABLE_REVERSE_PROXY": "true", "CATALOG_CACHE_ALLOW_CACHE_ABOVE_FREE_DISK_SPACE": "false", "CATALOG_CACHE_KEEP_FREE_DISK_SPACE": "0", "CATALOG_CACHE_MAX_SIZE": "12345678901234567890.125", "CUSTOM": "0"} {
		if got.Environment[key] != want {
			t.Errorf("mapping failed for %s", key)
		}
	}
	cfg.RootPassword = types.StringNull()
	cfg.EnvironmentVariables["ROOT_PASSWORD"] = types.StringValue(secret)
	got, err = ResolveServiceConfig(context.Background(), cfg, "fallback", "mac.local", true)
	if err != nil || got.Environment["ROOT_PASSWORD"] != secret {
		t.Fatal("environment password was not preserved")
	}
	cfg.EnvironmentVariables["ROOT_PASSWORD"] = types.StringValue("")
	if _, err = ResolveServiceConfig(context.Background(), cfg, "fallback", "mac.local", true); err == nil {
		t.Fatal("explicit empty password accepted")
	}
}

func TestConfigObjectRoundTripAndChanges(t *testing.T) {
	cfg := ParallelsDesktopDevopsConfigV3{Port: types.StringValue("9090"), Prefix: types.StringValue("/custom"), TLSPort: types.StringValue("9443")}
	obj := cfg.MapObject()
	if len(obj.Attributes()) != reflect.TypeOf(cfg).NumField() {
		t.Fatal("missing schema mappings")
	}
	if obj.Attributes()["port"] != cfg.Port || obj.Attributes()["tls_port"] != cfg.TLSPort || obj.Attributes()["prefix"] != cfg.Prefix {
		t.Fatal("incorrect object keys")
	}
	other := cfg
	other.EnabledModules = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("api"), types.StringValue("host")})
	if !ApiConfigHasChanges(context.Background(), &cfg, &other) {
		t.Fatal("module change missed")
	}
}

func TestEquivalentNumbersDoNotTriggerConfigurationReinstall(t *testing.T) {
	first := ParallelsDesktopDevopsConfigV3{CatalogCacheMaxSize: types.NumberValue(big.NewFloat(12.5))}
	second := ParallelsDesktopDevopsConfigV3{CatalogCacheMaxSize: types.NumberValue(big.NewFloat(12.5))}
	if ApiConfigHasChanges(context.Background(), &first, &second) {
		t.Fatal("equal numbers caused an unnecessary service restart")
	}
}
