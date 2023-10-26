package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-parallels/internal/clientmodels"
	"terraform-provider-parallels/internal/helpers"
	"terraform-provider-parallels/internal/ssh"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
)

type ParallelsServerClient struct {
	client *ssh.SshClient
	ctx    context.Context
}

func NewParallelsServerClient(ctx context.Context, client *ssh.SshClient) *ParallelsServerClient {
	return &ParallelsServerClient{
		client: client,
		ctx:    ctx,
	}
}

func (c *ParallelsServerClient) GetInfo() (*clientmodels.ParallelsServerInfo, error) {
	output, err := c.client.RunCommand("/usr/local/bin/prlsrvctl info --json")
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

func (c *ParallelsServerClient) GetVersion() (string, error) {
	parallelsInfo, err := c.GetInfo()
	if err != nil {
		return "", err
	}

	return parallelsInfo.Version, nil
}

func (c *ParallelsServerClient) RestartServer() error {
	restartServiceCmd := "/Applications/Parallels\\ Desktop.app/Contents/MacOS/Parallels\\ Service start && /Applications/Parallels\\ Desktop.app/Contents/MacOS/Parallels\\ Service start"
	_, err := c.client.RunCommand(restartServiceCmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) InstallDependencies() error {
	// Installing Brew
	_, err := c.client.RunCommand("/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
	if err != nil {
		return errors.New("Error running brew install command, error: " + err.Error())
	}

	// Installing Git
	_, err = c.client.RunCommand("/opt/homebrew/bin/brew install git")
	if err != nil {
		return errors.New("Error running git install command, error: " + err.Error())
	}

	// Installing Packer
	_, err = c.client.RunCommand("/opt/homebrew/bin/brew install packer")
	if err != nil {
		return errors.New("Error running packer install command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) UninstallDependencies() error {
	// Uninstalling Git
	_, err := c.client.RunCommand(" /opt/homebrew/bin/brew uninstall git")
	if err != nil {
		return errors.New("Error running git uninstall command, error: " + err.Error())
	}

	// Uninstalling Packer
	_, err = c.client.RunCommand("/opt/homebrew/bin/brew uninstall packer")
	if err != nil {
		return errors.New("Error running packer uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) InstallParallelsDesktop() error {
	// Installing parallels desktop using command line
	_, err := c.client.RunCommand("/opt/homebrew/bin/brew install parallels")
	if err != nil {
		return errors.New("Error running parallels install command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) UninstallParallelsDesktop() error {
	// Uninstalling parallels desktop using command line
	_, err := c.client.RunCommand("/opt/homebrew/bin/brew uninstall parallels")
	if err != nil {
		return errors.New("Error running parallels uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) GetLicense() (*ParallelsLicense, error) {
	output, err := c.client.RunCommand("/usr/local/bin/prlsrvctl info --json")
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

	parallelsLicense := ParallelsLicense{}
	parallelsLicense.FromClientModel(parallelsInfo.License)
	return &parallelsLicense, nil
}

func (c *ParallelsServerClient) InstallLicense(key string, username, password string) error {
	cmd := ""
	installLicense := fmt.Sprintf("/Applications/Parallels\\ Desktop.app/Contents/MacOS/prlsrvctl install-license --key %s --activate-online-immediately", key)
	if username != "" && password != "" {
		textPassword := fmt.Sprintf("echo %s >~/parallels_password.txt", password)
		loginCommand := fmt.Sprintf("/Applications/Parallels\\ Desktop.app/Contents/MacOS/prlsrvctl web-portal signin %s --read-passwd ~/parallels_password.txt", username)
		_, err := c.client.RunCommand(fmt.Sprintf("echo $PARALLELS_USER_PASSWORD >~/parallels_password.txt && /Applications/Parallels\\ Desktop.app/Contents/MacOS/prlsrvctl install-license --key %s --activate-online-immediately", key))
		if err != nil {
			return err
		}
		_, err = c.client.RunCommand(fmt.Sprintf("/Applications/Parallels\\ Desktop.app/Contents/MacOS/prlsrvctl install-license --key %s --activate-online-immediately", key))
		if err != nil {
			return err
		}
		cmd = fmt.Sprintf("%s && %s && %s", textPassword, loginCommand, installLicense)
	} else {
		cmd = installLicense
	}

	// Activate Parallels Desktop
	_, err := c.client.RunCommand(cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) DeactivateLicense() error {
	deactivateLicense := fmt.Sprintf("/Applications/Parallels\\ Desktop.app/Contents/MacOS/prlsrvctl deactivate-license --skip-network-errors")
	// Deactivate Parallels Desktop
	_, err := c.client.RunCommand(deactivateLicense)
	if err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) CompareLicenses(license string) (bool, error) {
	currentLicense, err := c.GetLicense()
	tflog.Info(c.ctx, "Current license: "+currentLicense.Key.ValueString())
	if err != nil {
		return false, err
	}

	if currentLicense == nil && license == "" {
		tflog.Info(c.ctx, "No license found")
		return true, nil
	}

	if currentLicense.Key.ValueString() == "" && license == "" {
		tflog.Info(c.ctx, "No license found1")
		return true, nil
	}

	currentLicenseKeyParts := strings.Split(currentLicense.Key.ValueString(), "-")
	licenseKeyParts := strings.Split(license, "-")
	if len(currentLicenseKeyParts) != len(licenseKeyParts) {
		tflog.Info(c.ctx, "License key parts not equal")
		return false, nil
	}
	if strings.EqualFold(currentLicenseKeyParts[0], licenseKeyParts[0]) &&
		strings.EqualFold(currentLicenseKeyParts[len(currentLicenseKeyParts)-1], licenseKeyParts[len(licenseKeyParts)-1]) {
		tflog.Info(c.ctx, "License key parts equal")
		return true, nil
	}

	tflog.Info(c.ctx, "License key parts not equal1")
	return false, nil
}

func (c *ParallelsServerClient) InstallApiService(license, version, port string) (string, error) {
	var releaseDetails clientmodels.GithubRelease
	var baseUrl string

	if version == "" || version == "latest" {
		tflog.Info(c.ctx, "PD Api version not specified, installing latest version")
		baseUrl = "https://api.github.com/repos/Parallels/pd-api-service/releases/latest"
	} else {
		tflog.Info(c.ctx, "PD Api version specified, installing version: "+version)
		baseUrl = "https://api.github.com/repos/Parallels/pd-api-service/releases/tags/" + version
	}
	if port == "" {
		port = "8080"
	}
	path := "/usr/local/bin"

	caller := helpers.NewHttpCaller(c.ctx)
	if _, err := caller.GetDataFromClient(baseUrl, nil, helpers.HttpCallerAuth{}, &releaseDetails); err != nil {
		return "", err
	}

	finalVersion := strings.ReplaceAll(releaseDetails.TagName, "v", "")
	tflog.Info(c.ctx, "PD Api Latest version: "+releaseDetails.TagName)

	tflog.Info(c.ctx, "Getting the url for the correct asset to download")
	// Getting the right asset
	var assetUrl string
	for _, asset := range releaseDetails.Assets {
		if strings.Contains(asset.Name, "darwin-amd64") {
			assetUrl = asset.BrowserDownloadURL
			break
		}
	}

	if assetUrl == "" {
		tflog.Error(c.ctx, "Error getting asset url")
		return "", errors.New("Error getting asset url")
	}

	tflog.Info(c.ctx, "Downloading the asset")
	// Downloading the asset
	_, err := c.client.RunCommand(fmt.Sprintf("curl -L -o /tmp/pd-api-service.tar.gz %s", assetUrl))
	if err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Extracting the asset")
	_, err = c.client.RunCommand("sudo tar -xzf /tmp/pd-api-service.tar.gz -C " + path)
	if err != nil {
		return "", err
	}
	_, err = c.client.RunCommand("sudo rm -f /tmp/pd-api-service.tar.gz")
	if err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Installing the service")
	plist, err := getApiServicePlist(path, port)
	if err != nil {
		tflog.Error(c.ctx, "Error getting plist template")
		return "", err
	}

	tflog.Info(c.ctx, "Creating plist file")

	echoCmd := fmt.Sprintf("sudo echo '%s' > /tmp/com.parallels.api-service.plist", plist)
	mvCmd := "sudo mv /tmp/com.parallels.api-service.plist /Library/LaunchDaemons/com.parallels.api-service.plist"
	rmCmd := "sudo rm -f /tmp/com.parallels.api-service.plist"
	_, err = c.client.RunCommand(fmt.Sprintf("%s && %s && %s", echoCmd, mvCmd, rmCmd))
	if err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Starting server")
	chownCmd := fmt.Sprintf("sudo chown root:wheel /Library/LaunchDaemons/com.parallels.api-service.plist")
	chmodCmd := fmt.Sprintf("sudo chmod 644 /Library/LaunchDaemons/com.parallels.api-service.plist")
	unloadCmd := fmt.Sprintf("sudo launchctl unload /Library/LaunchDaemons/com.parallels.api-service.plist")
	loadCmd := fmt.Sprintf("sudo launchctl load /Library/LaunchDaemons/com.parallels.api-service.plist")
	_, err = c.client.RunCommand(fmt.Sprintf("%s && %s && %s && %s", chownCmd, chmodCmd, unloadCmd, loadCmd))
	if err != nil {
		return "", err
	}

	password := license
	changePassCmd := fmt.Sprintf("sudo %s/pd-api-service --update-root-pass --password=%s", path, password)
	cmd := fmt.Sprintf("%s && %s && %s > /tmp/change_password.log", unloadCmd, changePassCmd, loadCmd)
	tflog.Info(c.ctx, "Changing root password cmd: "+cmd)
	// change the root password for the license one
	out, err := c.client.RunCommand(cmd)
	if err != nil {
		return "", errors.New("Error changing root password, out: " + out + " error: " + err.Error())
	}

	tflog.Info(c.ctx, "Done")
	return finalVersion, nil
}

func (c *ParallelsServerClient) UninstallApiService() error {
	unloadCmd := fmt.Sprintf("sudo launchctl unload /Library/LaunchDaemons/com.parallels.api-service.plist")
	_, err := c.client.RunCommand(unloadCmd)
	if err != nil {
		return err
	}

	rmPlistCmd := "sudo rm -f /tmp/com.parallels.api-service.plist"
	rmExecCmd := "sudo rm -f /usr/local/bin/pd-api-service"
	rmLogsCmd := "sudo rm -f /tmp/api-service.job.*"
	_, err = c.client.RunCommand(fmt.Sprintf("%s && %s && %s", rmPlistCmd, rmExecCmd, rmLogsCmd))
	if err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) GetApiVersion() (string, error) {
	output, err := c.client.RunCommand("/usr/local/bin/pd-api-service --version")
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *ParallelsServerClient) GenerateDefaultRootPassword() (string, error) {
	info, err := c.GetInfo()
	if err != nil {
		return "", err
	}

	key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(info.License.Key, "-", ""), "*", ""))
	hid := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(info.HardwareID, "-", ""), "{", ""), "}", ""))

	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", key, hid)))

	return encoded, nil
}
