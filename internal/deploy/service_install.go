package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"terraform-provider-parallels-desktop/internal/deploy/models"

	"gopkg.in/yaml.v3"
)

const installerRevision = "797a00b5b89aed885f870fda142c22d31c4f9a9a"
const runtimeConfigPath = "/etc/prl-devops-service/prldevops_config.yaml"

var releaseVersion = regexp.MustCompile(`^(?:release-)?v?(\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?)$`)

func normalizeRelease(value string) (string, error) {
	m := releaseVersion.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return "", errors.New("DevOps Service version must be a concrete semantic version")
	}
	return m[1], nil
}

func installedRelease(output string) (string, error) {
	output = strings.TrimSpace(output)
	if strings.Contains(output, " version ") {
		fields := strings.Fields(strings.SplitN(output, " version ", 2)[1])
		if len(fields) == 0 {
			return "", errors.New("missing installed version")
		}
		return normalizeRelease(fields[0])
	}
	return normalizeRelease(output)
}

func resolveRelease(ctx context.Context, beta bool) (string, error) {
	endpoint := "https://api.github.com/repos/Parallels/prl-devops-service/releases/latest"
	if beta {
		endpoint = "https://api.github.com/repos/Parallels/prl-devops-service/releases?per_page=100"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not resolve DevOps release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("release lookup returned HTTP %d", resp.StatusCode)
	}
	type release struct {
		Tag        string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	dec := json.NewDecoder(io.LimitReader(resp.Body, 2<<20))
	if !beta {
		var r release
		if err := dec.Decode(&r); err != nil {
			return "", err
		}
		return normalizeRelease(r.Tag)
	}
	var releases []release
	if err := dec.Decode(&releases); err != nil {
		return "", err
	}
	for _, r := range releases {
		if !r.Draft && r.Prerelease {
			return normalizeRelease(r.Tag)
		}
	}
	return "", errors.New("no prerelease available")
}

func (c *DevOpsServiceClient) InstallDevOpsService(ctx context.Context, license string, input models.ParallelsDesktopDevopsConfigV3) (string, error) {
	cfg, err := models.ResolveServiceConfig(ctx, &input, license, "localhost", false)
	if err != nil {
		return "", err
	}
	return c.installConfiguredService(ctx, cfg)
}

func (c *DevOpsServiceClient) installConfiguredService(ctx context.Context, cfg models.EffectiveServiceConfig) (string, error) {
	binary, found, err := c.findPath(ctx, "prldevops")
	if err != nil {
		return "", err
	}
	requested := cfg.Version
	if requested != "" && requested != "latest" {
		requested, err = normalizeRelease(requested)
		if err != nil {
			return "", err
		}
	}
	if found {
		version, err := c.GetDevOpsVersion(ctx)
		if err != nil {
			return "", err
		}
		actual, err := installedRelease(version)
		if err != nil {
			return "", fmt.Errorf("cannot verify installed DevOps Service version: %w", err)
		}
		if requested != "" && requested != "latest" && actual != requested {
			return "", fmt.Errorf("installed DevOps Service version %s differs from requested %s; upgrade the binary explicitly before applying", actual, requested)
		}
		if cfg.Beta && !strings.Contains(actual, "-") {
			return "", errors.New("installed DevOps Service is stable; install the desired prerelease before applying")
		}
		requested = actual
	} else if requested == "" || requested == "latest" {
		requested, err = resolveRelease(ctx, cfg.Beta)
		if err != nil {
			return "", err
		}
	}
	// This adapter was reviewed against module-based releases 1.0.4 and 1.1.0.
	if err := validateReleaseSupport(requested); err != nil {
		return "", err
	}

	if !found {
		script := `set -euo pipefail
curl -fsSL "https://raw.githubusercontent.com/Parallels/prl-devops-service/` + installerRevision + `/scripts/install.sh" | /bin/bash -s -- --no-service --version "$1"`
		if _, err := c.client.RunCommandContext(ctx, "/bin/bash", []string{"-c", script, "install-devops", requested}); err != nil {
			return "", err
		}
		delete(c.paths, "prldevops")
		binary, err = c.requirePath(ctx, "prldevops")
		if err != nil {
			return "", err
		}
		version, err := c.GetDevOpsVersion(ctx)
		if err != nil {
			return "", err
		}
		actual, err := installedRelease(version)
		if err != nil || actual != requested {
			return "", fmt.Errorf("installed binary did not match the resolved release %s", requested)
		}
	}
	// Read the canonical configuration before invoking the legacy installer, which overwrites it.
	encodedExisting, err := c.client.RunCommandContext(ctx, "sudo", []string{"-n", "/bin/bash", "-c", `if [ -f "$1" ]; then /usr/bin/base64 < "$1"; fi`, "read-devops-config", runtimeConfigPath})
	if err != nil {
		return "", err
	}
	existing, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedExisting))
	if err != nil {
		return "", errors.New("could not decode canonical DevOps configuration snapshot")
	}
	runtime, err := mergeRuntimeConfig(existing, cfg.Environment)
	if err != nil {
		return "", err
	}
	bootstrap := map[string]interface{}{"port": cfg.Environment["API_PORT"], "prefix": cfg.Prefix, "root_password": cfg.Environment["ROOT_PASSWORD"], "enabled_modules": cfg.Environment["ENABLED_MODULES"], "hmac_secret": cfg.Environment["HMAC_SECRET"], "encryption_rsa_key": cfg.Environment["ENCRYPTION_PRIVATE_KEY"], "enable_tls": cfg.Protocol == "https", "tls_port": cfg.Environment["TLS_PORT"], "tls_certificate": cfg.Environment["TLS_CERTIFICATE"], "tls_private_key": cfg.Environment["TLS_PRIVATE_KEY"]}
	bootstrapJSON, err := json.Marshal(bootstrap)
	if err != nil {
		return "", err
	}
	payload := base64.StdEncoding.EncodeToString(bootstrapJSON) + "\n" + base64.StdEncoding.EncodeToString(runtime) + "\n" + fmt.Sprintf("%x\n", sha256.Sum256(existing))
	if _, err := c.client.RunCommandInput(ctx, "sudo", []string{"-n", "/bin/bash", "-c", configureServiceScript, "configure-devops", binary}, strings.NewReader(payload)); err != nil {
		return "", fmt.Errorf("applying canonical DevOps configuration failed (the transaction attempts to restore the previous configuration): %w", err)
	}
	return requested, nil
}

func mergeRuntimeConfig(existing []byte, env map[string]string) ([]byte, error) {
	var root map[string]interface{}
	if len(existing) > 0 {
		if err := yaml.Unmarshal(existing, &root); err != nil {
			return nil, errors.New("existing canonical DevOps configuration is invalid YAML")
		}
	}
	if root == nil {
		root = map[string]interface{}{}
	}
	root["environment"] = env
	return yaml.Marshal(root)
}

// Secrets are supplied only on stdin. The transaction never deletes the database.
// Configuration-only updates keep the binary and database, and refresh launchd.
//
//nolint:dupword // Repeated fi tokens are required Bash control-flow terminators.
const configureServiceScript = `set -euo pipefail
umask 077
config_dir=/etc/prl-devops-service
config_file="$config_dir/prldevops_config.yaml"
plist=/Library/LaunchDaemons/com.parallels.prl-devops-service.plist
label=com.parallels.devops-service
mkdir -p "$config_dir"
mkdir "$config_dir/.terraform-config-lock" || { echo 'another configuration transaction is active' >&2; exit 1; }
trap 'rmdir "$config_dir/.terraform-config-lock"' EXIT
work=$(mktemp -d "$config_dir/.terraform-config.XXXXXXXX")
success=false
had_config=false
had_plist=false
was_loaded=false
snapshots_ready=false
cleanup() {
 status=$?
 trap - EXIT HUP INT TERM
 if [ "$success" != true ] && [ "$snapshots_ready" = true ]; then
   if launchctl list "$label" >/dev/null 2>&1; then launchctl unload "$plist" || true; fi
   if [ "$had_config" = true ]; then mv -f "$work/previous.yaml" "$config_file"; else rm -f "$config_file"; fi
   if [ "$had_plist" = true ]; then mv -f "$work/previous.plist" "$plist"; else rm -f "$plist"; fi
   if [ "$was_loaded" = true ]; then launchctl load "$plist" || echo 'failed to restore launchd job' >&2; fi
 fi
 rm -rf "$work"
 rmdir "$config_dir/.terraform-config-lock"
 exit "$status"
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM
if [ -f "$config_file" ]; then cp -p "$config_file" "$work/previous.yaml"; had_config=true; fi
if [ -f "$plist" ]; then cp -p "$plist" "$work/previous.plist"; had_plist=true; fi
if launchctl list "$label" >/dev/null 2>&1; then was_loaded=true; fi
snapshots_ready=true
IFS= read -r bootstrap
IFS= read -r runtime
IFS= read -r expected_hash
actual_hash=$(if [ -f "$config_file" ]; then /usr/bin/shasum -a 256 "$config_file"; else printf '' | /usr/bin/shasum -a 256; fi)
if [ "${actual_hash%% *}" != "$expected_hash" ]; then echo 'canonical configuration changed concurrently; retry apply' >&2; exit 1; fi
printf '%s' "$bootstrap" | /usr/bin/base64 -D > "$work/install.json"
printf '%s' "$runtime" | /usr/bin/base64 -D > "$work/runtime.yaml"
# Rebuild the service definition to remove stale command-line/environment overrides.
# The native service installer leaves the binary and database intact.
if launchctl list "$label" >/dev/null 2>&1; then launchctl unload "$plist"; fi
rm -f "$plist"
if [ -f "$config_file" ]; then chmod 600 "$config_file"; fi
"$1" install service --file="$work/install.json"
if launchctl list "$label" >/dev/null 2>&1; then launchctl unload "$plist"; fi
chown root:wheel "$work/runtime.yaml"
chmod 600 "$work/runtime.yaml"
mv -f "$work/runtime.yaml" "$config_file"
launchctl load "$plist"
success=true
`
