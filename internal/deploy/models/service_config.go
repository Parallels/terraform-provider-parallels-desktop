package models

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EffectiveServiceConfig is derived without modifying Terraform's configured values.
// Environment is sensitive and must never be included in logs or diagnostics.
type EffectiveServiceConfig struct {
	Environment                                     map[string]string
	Endpoint, Host, Port, Protocol, Prefix, Version string
	Beta                                            bool
}

func ResolveServiceConfig(ctx context.Context, input *ParallelsDesktopDevopsConfigV3, license, host string, registration bool) (EffectiveServiceConfig, error) {
	c := ParallelsDesktopDevopsConfigV3{}
	if input != nil {
		c = *input
	}
	out := EffectiveServiceConfig{Environment: map[string]string{}, Host: strings.Trim(host, "[]"), Version: c.DevOpsVersion.ValueString(), Beta: c.UseLatestBeta.ValueBool()}
	env := out.Environment
	// Fail before any remote side effects when apply still contains unknown values.
	v := reflect.ValueOf(c)
	for i := range v.NumField() {
		if a, ok := v.Field(i).Interface().(attr.Value); ok && a.IsUnknown() {
			return out, fmt.Errorf("api_config.%s must be known before deployment", v.Type().Field(i).Tag.Get("tfsdk"))
		}
	}
	for k, v := range c.EnvironmentVariables {
		if v.IsUnknown() || v.IsNull() {
			return out, fmt.Errorf("environment_variables[%s] must have a known string value", k)
		}
		env[k] = v.ValueString()
	}
	for old, key := range map[string]string{"PRL_DEVOPS_LOG_TO_FILE": "LOG_TO_FILE", "PRL_DEVOPS_LOG_FILE_PATH": "LOG_FILE_PATH"} {
		if value, ok := env[old]; ok {
			if other, exists := env[key]; exists && other != value {
				return out, fmt.Errorf("conflicting environment aliases %s and %s", old, key)
			}
			env[key] = value
			delete(env, old)
		}
	}
	set := func(key, value string) error {
		if existing, ok := env[key]; ok && existing != value {
			equal := false
			switch key {
			case "API_PORT", "TLS_PORT", "CATALOG_CACHE_KEEP_FREE_DISK_SPACE", "CATALOG_CACHE_MAX_SIZE":
				equal = equalRatStrings(existing, value)
			case "LOG_LEVEL":
				equal = strings.EqualFold(existing, value)
			case "API_PREFIX":
				equal = strings.TrimRight(existing, "/") == strings.TrimRight(value, "/")
			}
			if !equal {
				return fmt.Errorf("api_config conflicts with environment_variables[%s]", key)
			}
		}
		env[key] = value
		return nil
	}
	stringsMap := map[string]types.String{"API_PORT": c.Port, "API_PREFIX": c.Prefix, "ROOT_PASSWORD": c.RootPassword, "HMAC_SECRET": c.HmacSecret, "ENCRYPTION_PRIVATE_KEY": c.EncryptionRsaKey, "LOG_LEVEL": c.LogLevel, "TLS_PORT": c.TLSPort, "TLS_CERTIFICATE": c.TLSCertificate, "TLS_PRIVATE_KEY": c.TLSPrivateKey, "TOKEN_DURATION_MINUTES": c.TokenDurationMinutes, "SYSTEM_RESERVED_MEMORY": c.SystemReservedMemory, "SYSTEM_RESERVED_CPU": c.SystemReservedCpu, "SYSTEM_RESERVED_DISK": c.SystemReservedDisk, "LOG_FILE_PATH": c.LogPath}
	for key, value := range stringsMap {
		if !value.IsNull() {
			if err := set(key, value.ValueString()); err != nil {
				return out, err
			}
		}
	}
	boolMap := map[string]types.Bool{"TLS_ENABLED": c.EnableTLS, "DISABLE_CATALOG_CACHING": c.DisableCatalogCaching, "CATALOG_CACHE_ALLOW_CACHE_ABOVE_FREE_DISK_SPACE": c.CatalogCacheAllowCacheAboveKeepFreeDiskSpace, "DISABLE_CATALOG_PROVIDER_STREAMING": c.DisableCatalogCachingStream, "USE_ORCHESTRATOR_RESOURCES": c.UseOrchestratorResources, "LOG_TO_FILE": c.EnableLogging}
	if !c.EnablePortForwarding.IsNull() {
		boolMap["DISABLE_REVERSE_PROXY"] = types.BoolValue(!c.EnablePortForwarding.ValueBool())
	}
	for key, value := range boolMap {
		if existing, ok := env[key]; ok {
			b, err := strconv.ParseBool(existing)
			if err != nil {
				return out, fmt.Errorf("%s must be a boolean", key)
			}
			env[key] = strconv.FormatBool(b)
		}
		if !value.IsNull() {
			if err := set(key, strconv.FormatBool(value.ValueBool())); err != nil {
				return out, err
			}
		}
	}
	for key, value := range map[string]types.Number{"CATALOG_CACHE_KEEP_FREE_DISK_SPACE": c.CatalogCacheKeepFreeDiskSpace, "CATALOG_CACHE_MAX_SIZE": c.CatalogCacheMaxSize} {
		if !value.IsNull() {
			if value.ValueBigFloat().Sign() < 0 {
				return out, fmt.Errorf("%s cannot be negative", key)
			}
			if err := set(key, value.ValueBigFloat().Text('f', -1)); err != nil {
				return out, err
			}
		}
	}
	for key, value := range map[string]string{"API_PORT": "8080", "TLS_PORT": "8443", "API_PREFIX": "/api", "ROOT_PASSWORD": license, "TLS_ENABLED": "false"} {
		if _, ok := env[key]; !ok {
			env[key] = value
		}
	}
	for _, key := range []string{"API_PORT", "TLS_PORT"} {
		n, err := strconv.Atoi(env[key])
		if err != nil || n < 1 || n > 65535 {
			return out, fmt.Errorf("%s must be a port between 1 and 65535", key)
		}
		env[key] = strconv.Itoa(n)
	}
	prefix := strings.TrimRight(env["API_PREFIX"], "/")
	if prefix == "" || !strings.HasPrefix(prefix, "/") || strings.ContainsAny(prefix, "?#\\") || strings.Contains(prefix, "//") {
		return out, errors.New("API_PREFIX must be an absolute URL path")
	}
	env["API_PREFIX"] = prefix
	normalizeModules := func(value string) (string, error) {
		allowed := map[string]bool{"api": true, "host": true, "orchestrator": true, "catalog": true, "catalog_manager": true, "cache": true, "reverse_proxy": true, "cors": true}
		seen := map[string]bool{}
		for _, m := range strings.Split(value, ",") {
			m = strings.TrimSpace(m)
			if !allowed[m] {
				return "", fmt.Errorf("invalid enabled module %q", m)
			}
			seen[m] = true
		}
		list := []string{}
		for m := range seen {
			list = append(list, m)
		}
		sort.Strings(list)
		return strings.Join(list, ","), nil
	}
	modules := ""
	mergeModules := func(value string) error {
		n, err := normalizeModules(value)
		if err != nil {
			return err
		}
		if modules != "" && modules != n {
			return errors.New("mode, enabled_modules and ENABLED_MODULES must agree")
		}
		modules = n
		return nil
	}
	if value, ok := env["ENABLED_MODULES"]; ok {
		if err := mergeModules(value); err != nil {
			return out, err
		}
	}
	if !c.EnabledModules.IsNull() {
		var list []string
		if d := c.EnabledModules.ElementsAs(ctx, &list, false); d.HasError() {
			return out, errors.New("enabled_modules must contain known strings")
		}
		if err := mergeModules(strings.Join(list, ",")); err != nil {
			return out, err
		}
	}
	mode := c.Mode.ValueString()
	if legacy, ok := env["MODE"]; ok {
		if mode != "" && mode != legacy {
			return out, errors.New("mode conflicts with MODE")
		}
		mode = legacy
		delete(env, "MODE")
	}
	if mode != "" {
		value := "api,host"
		switch mode {
		case "api":
		case "catalog", "orchestrator":
			value += "," + mode
		default:
			return out, errors.New("invalid service mode")
		}
		if err := mergeModules(value); err != nil {
			return out, err
		}
	}
	if modules == "" {
		modules = "api,host"
	}
	env["ENABLED_MODULES"] = modules
	hasModule := func(m string) bool {
		for _, v := range strings.Split(modules, ",") {
			if v == m {
				return true
			}
		}
		return false
	}
	if !hasModule("api") || (registration && !hasModule("host")) {
		return out, errors.New("deployment requires the api module; registration also requires the host module")
	}
	if registration && env["ROOT_PASSWORD"] == "" {
		return out, errors.New("registration requires root_password or ROOT_PASSWORD (or a provider license fallback)")
	}
	if out.Beta && out.Version != "" && out.Version != "latest" {
		return out, errors.New("use_latest_beta cannot be combined with an explicit devops_version")
	}
	out.Protocol = "http"
	out.Port = env["API_PORT"]
	if env["TLS_ENABLED"] == "true" {
		out.Protocol = "https"
		out.Port = env["TLS_PORT"]
		if env["TLS_CERTIFICATE"] == "" || env["TLS_PRIVATE_KEY"] == "" {
			return out, errors.New("TLS requires TLS_CERTIFICATE and TLS_PRIVATE_KEY")
		}
	}
	if out.Host == "" || strings.ContainsAny(out.Host, "/?#@") || (strings.Contains(out.Host, ":") && net.ParseIP(out.Host) == nil) {
		return out, errors.New("deployment host must be a hostname or IP address")
	}
	out.Prefix = prefix
	out.Endpoint = (&url.URL{Scheme: out.Protocol, Host: net.JoinHostPort(out.Host, out.Port), Path: prefix}).String()
	return out, nil
}

func equalRatStrings(a, b string) bool {
	const maxNumericInputLength = 256
	if len(a) > maxNumericInputLength || len(b) > maxNumericInputLength {
		return false
	}
	left, leftOK := new(big.Rat).SetString(a)   // #nosec G113 -- inputs are length-limited above.
	right, rightOK := new(big.Rat).SetString(b) // #nosec G113 -- inputs are length-limited above.
	return leftOK && rightOK && left.Cmp(right) == 0
}
