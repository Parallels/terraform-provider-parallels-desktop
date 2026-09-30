package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"terraform-provider-parallels-desktop/internal/clientmodels"
	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/interfaces"
	"terraform-provider-parallels-desktop/internal/localclient"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
)

type DevOpsServiceClient struct {
	client interfaces.ContextCommandClient
	paths  map[string]string
}

func NewDevOpsServiceClient(ctx context.Context, client interfaces.ContextCommandClient) *DevOpsServiceClient {
	return &DevOpsServiceClient{
		client: client,
		paths:  make(map[string]string),
	}
}

func (c *DevOpsServiceClient) GetInfo(ctx context.Context) (*clientmodels.ParallelsServerInfo, error) {
	cmd, err := c.requirePath(ctx, "prlsrvctl")
	if err != nil {
		return nil, err
	}
	arguments := []string{"info", "--json"}
	output, err := c.client.RunCommandContext(ctx, cmd, arguments)
	output = strings.ReplaceAll(output, "This feature is not available in this edition of Parallels Desktop. \n", "")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, errors.New("empty output")
	}

	var parallelsInfo clientmodels.ParallelsServerInfo
	err = json.Unmarshal([]byte(output), &parallelsInfo)
	if err != nil {
		return nil, err
	}

	return &parallelsInfo, nil
}

func (c *DevOpsServiceClient) GetVersion(ctx context.Context) (string, error) {
	parallelsInfo, err := c.GetInfo(ctx)
	if err != nil {
		return "", err
	}

	return parallelsInfo.Version, nil
}

func (c *DevOpsServiceClient) RestartServer(ctx context.Context) error {
	_, err := c.client.RunCommandContext(ctx, "/Applications/Parallels Desktop.app/Contents/MacOS/Parallels Service", []string{"start"})
	return err
}

func (c *DevOpsServiceClient) GetLicense(ctx context.Context) (*models.ParallelsDesktopLicense, error) {
	cmd, err := c.requirePath(ctx, "prlsrvctl")
	if err != nil {
		return nil, err
	}
	arguments := []string{"info", "--json"}
	output, err := c.client.RunCommandContext(ctx, cmd, arguments)
	output = strings.ReplaceAll(output, "This feature is not available in this edition of Parallels Desktop. \n", "")
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, errors.New("empty output")
	}

	var parallelsInfo clientmodels.ParallelsServerInfo
	err = json.Unmarshal([]byte(output), &parallelsInfo)
	if err != nil {
		return nil, err
	}

	parallelsLicense := models.ParallelsDesktopLicense{}
	parallelsLicense.FromClientModel(parallelsInfo.License)
	return &parallelsLicense, nil
}

func (c *DevOpsServiceClient) InstallLicense(ctx context.Context, key, username, password string) error {
	cmd, err := c.requirePath(ctx, "prlsrvctl")
	if err != nil {
		return err
	}
	if username != "" && password != "" {
		// A private, unique file is required by prlsrvctl; credentials travel on stdin.
		script := `set -eu
umask 077
password_file=$(mktemp "${TMPDIR:-/tmp}/parallels-password.XXXXXXXX")
trap 'rm -f "$password_file"' EXIT
cat > "$password_file"
"$1" web-portal signin "$2" --read-passwd "$password_file"
`
		if _, err := c.client.RunCommandInput(ctx, "/bin/bash", []string{"-c", script, "license-signin", cmd, username}, strings.NewReader(password+"\n")); err != nil {
			return err
		}
	}
	_, err = c.client.RunCommandContext(ctx, cmd, []string{"install-license", "--key", key, "--activate-online-immediately"})
	return err
}

func (c *DevOpsServiceClient) DeactivateLicense(ctx context.Context) error {
	// For local clients, never deactivate the host's Parallels Desktop license.
	// We skip license installation during create (host is already licensed),
	// so we must also skip deactivation during destroy.
	if _, isLocal := c.client.(*localclient.LocalClient); isLocal {
		tflog.Info(ctx, "Skipping license deactivation for local client — host license is not managed by Terraform")
		return nil
	}

	cmd, err := c.requirePath(ctx, "prlsrvctl")
	if err != nil {
		return err
	}
	arguments := []string{"deactivate-license", "--skip-network-errors"}

	if _, err := c.client.RunCommandContext(ctx, cmd, arguments); err != nil {
		return err
	}

	return nil
}

func (c *DevOpsServiceClient) CompareLicenses(ctx context.Context, license string) (bool, error) {
	currentLicense, err := c.GetLicense(ctx)
	if err != nil || currentLicense == nil {
		return false, err
	}

	if currentLicense.Key.IsUnknown() || currentLicense.Key.IsNull() {
		tflog.Info(ctx, "Current license: "+currentLicense.Key.ValueString())
	} else {
		tflog.Info(ctx, "Current license key is nil")
	}

	if license == "" {
		tflog.Info(ctx, "No license found")
		return true, nil
	}

	if currentLicense.Key.ValueString() == "" && license == "" {
		tflog.Info(ctx, "No license found1")
		return true, nil
	}

	currentLicenseKeyParts := strings.Split(currentLicense.Key.ValueString(), "-")
	licenseKeyParts := strings.Split(license, "-")
	if len(currentLicenseKeyParts) != len(licenseKeyParts) {
		tflog.Info(ctx, "License key parts not equal")
		return false, nil
	}
	if strings.EqualFold(currentLicenseKeyParts[0], licenseKeyParts[0]) &&
		strings.EqualFold(currentLicenseKeyParts[len(currentLicenseKeyParts)-1], licenseKeyParts[len(licenseKeyParts)-1]) {
		tflog.Info(ctx, "License key parts equal")
		return true, nil
	}

	tflog.Info(ctx, "License key parts not equal1")
	return false, nil
}

func (c *DevOpsServiceClient) UninstallDevOpsService(ctx context.Context) error {
	tflog.Info(ctx, "Uninstalling the Parallels Desktop DevOps Service")

	devopsPath, _, err := c.findPath(ctx, "prldevops")
	if err != nil {
		return err
	}
	defer delete(c.paths, "prldevops")
	if devopsPath == "" {
		tflog.Info(ctx, "prldevops binary not found — nothing to uninstall")
		return nil
	}

	if _, isLocal := c.client.(*localclient.LocalClient); isLocal {
		return c.uninstallDevOpsServiceLocal(ctx, devopsPath)
	}

	cmd := "/bin/bash"
	arguments := []string{"-c", "curl -fsSL https://raw.githubusercontent.com/Parallels/prl-devops-service/797a00b5b89aed885f870fda142c22d31c4f9a9a/scripts/install.sh | bash -s -- --uninstall"}

	_, err = c.client.RunCommandContext(ctx, cmd, arguments)
	if err != nil {
		return err
	}

	return nil
}

// uninstallDevOpsServiceLocal performs a best-effort cleanup of the DevOps
// service on the local machine. It does NOT use curl from the internet.
// Steps:
//  1. Stop & unregister the launchd service via "sudo prldevops uninstall service"
//  2. Remove the prldevops binary
//  3. Remove config files
func (c *DevOpsServiceClient) uninstallDevOpsServiceLocal(ctx context.Context, devopsPath string) error {
	var warnings []string

	// Step 1: Unregister the launchd service (requires sudo)
	tflog.Info(ctx, "Unregistering prldevops launchd service")
	_, err := c.client.RunCommandContext(ctx, "sudo", []string{"-n", devopsPath, "uninstall", "service"})
	if err != nil {
		warnings = append(warnings, "Failed to unregister launchd service: "+err.Error())
		tflog.Warn(ctx, "Failed to unregister prldevops service (sudo may not be cached): "+err.Error())
	}

	// Step 2: Remove the prldevops binary
	tflog.Info(ctx, "Removing prldevops binary at "+devopsPath)
	_, err = c.client.RunCommandContext(ctx, "rm", []string{"-f", devopsPath})
	if err != nil {
		// Try with sudo in case the binary is in a protected location
		_, err2 := c.client.RunCommandContext(ctx, "sudo", []string{"-n", "rm", "-f", devopsPath})
		if err2 != nil {
			warnings = append(warnings, "Failed to remove prldevops binary at "+devopsPath+": "+err2.Error())
		}
	}

	// Step 3: Remove config file if it exists
	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		configPath := filepath.Join(homeDir, ".parallels-devops-service.json")
		tflog.Info(ctx, "Removing config file at "+configPath)
		_, _ = c.client.RunCommandContext(ctx, "rm", []string{"-f", configPath})
	}

	if len(warnings) > 0 {
		return fmt.Errorf("partial cleanup — manual steps may be needed: %s", strings.Join(warnings, "; "))
	}

	tflog.Info(ctx, "DevOps service uninstalled successfully (local)")
	return nil
}

func (c *DevOpsServiceClient) GetDevOpsVersion(ctx context.Context) (string, error) {
	devopsPath, err := c.requirePath(ctx, "prldevops")
	if err != nil {
		return "", err
	}
	output, err := c.client.RunCommandContext(ctx, devopsPath, []string{"--version"})
	if err != nil {
		return "", err
	}
	return installedRelease(output)
}

func (c *DevOpsServiceClient) GetPackerVersion(ctx context.Context) (string, error) {
	cmd, err := c.requirePath(ctx, "packer")
	if err != nil {
		return "", err
	}
	arguments := []string{"--version"}
	output, err := c.client.RunCommandContext(ctx, cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *DevOpsServiceClient) GetVagrantVersion(ctx context.Context) (string, error) {
	cmd, err := c.requirePath(ctx, "vagrant")
	if err != nil {
		return "", err
	}
	arguments := []string{"--version"}
	output, err := c.client.RunCommandContext(ctx, cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "Vagrant  ", ""), nil
}

func (c *DevOpsServiceClient) GetGitVersion(ctx context.Context) (string, error) {
	cmd, err := c.requirePath(ctx, "git")
	if err != nil {
		return "", err
	}
	arguments := []string{"--version"}
	output, err := c.client.RunCommandContext(ctx, cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "git version ", ""), nil
}

func (c *DevOpsServiceClient) GenerateDefaultRootPassword(ctx context.Context) (string, error) {
	info, err := c.GetInfo(ctx)
	if err != nil {
		return "", err
	}

	key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(info.License.Key, "-", ""), "*", ""))
	hid := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(info.HardwareID, "-", ""), "{", ""), "}", ""))

	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", key, hid)))

	return encoded, nil
}
