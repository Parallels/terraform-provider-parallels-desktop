package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"terraform-provider-parallels-desktop/internal/clientmodels"
	"terraform-provider-parallels-desktop/internal/helpers"
	"terraform-provider-parallels-desktop/internal/interfaces"
	"terraform-provider-parallels-desktop/internal/localclient"
	"terraform-provider-parallels-desktop/internal/ssh"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
)

var (
	installPath     = "/usr/local/bin"
	executableNames = []string{"prldevops", "pd-api-service"}
)

type ParallelsServerClient struct {
	client interfaces.CommandClient
	ctx    context.Context
}

func NewParallelsServerClient(ctx context.Context, client interfaces.CommandClient) *ParallelsServerClient {
	return &ParallelsServerClient{
		client: client,
		ctx:    ctx,
	}
}

func (c *ParallelsServerClient) GetInfo() (*clientmodels.ParallelsServerInfo, error) {
	cmd := c.findPath("prlsrvctl")
	arguments := []string{"info", "--json"}
	output, err := c.client.RunCommand(cmd, arguments)
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
	cmd := "/Applications/Parallels\\ Desktop.app/Contents/MacOS/Parallels\\ Service"
	arguments := []string{"start"}
	_, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return err
	}

	_, err = c.client.RunCommand(cmd, arguments)
	if err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) InstallDependencies() error {
	// Installing Brew
	var cmd string
	var brewPath string
	var arguments []string

	_, ok := c.client.(*localclient.LocalClient)
	if !ok {
		brewPath = c.findPath("brew")
		if brewPath == "" {
			cmd = "/bin/bash"
			arguments = []string{"-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""}
			_, err := c.client.RunCommand(cmd, arguments)
			if err != nil {
				return errors.New("Error running brew install command, error: " + err.Error())
			}
		}
	}

	if c.findPath("brew") == "" {
		return errors.New("Error running brew install command, error: brew not found")
	}

	// Installing Git
	cmd = brewPath
	gitPresent := c.findPath("git")
	if gitPresent == "" {
		arguments = []string{"install", "git"}
		out, err := c.client.RunCommand(cmd, arguments)
		tflog.Info(c.ctx, out)
		if err != nil {
			return errors.New("Error running git install command, error: " + err.Error())
		}
	}

	// Installing Packer
	cmd = brewPath
	packerPresent := c.findPath("packer")
	if packerPresent == "" {
		arguments = []string{"install", "packer"}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running packer install command, error: " + err.Error())
		}
	}

	// Installing Vagrant
	if !ok {
		cmd = brewPath
		vagrantPresent := c.findPath("vagrant")
		if vagrantPresent == "" {
			arguments = []string{"install", "vagrant"}
			out, err := c.client.RunCommand(cmd, arguments)
			if err != nil {
				tflog.Info(c.ctx, "Vagrant install output: "+out)
				return errors.New("Error running vagrant install command, error: " + err.Error())
			}
		}

		// Installing Vagrant Parallels Plugin
		cmd = "/usr/local/bin/vagrant"
		arguments = []string{"plugin", "install", "vagrant-parallels"}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running  plugin install command, error: " + err.Error())
		}
	}

	return nil
}

func (c *ParallelsServerClient) UninstallDependencies() error {
	// Uninstalling Git
	cmd := c.findPath("brew")
	if c.findPath("brew") == "" {
		return nil
	}

	gitPresent := c.findPath("git")
	if gitPresent != "" {
		arguments := []string{"uninstall", "git"}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running git uninstall command, error: " + err.Error())
		}
	}

	// Uninstalling Packer
	packerPresent := c.findPath("packer")
	if packerPresent != "" {
		arguments := []string{"uninstall", "packer"}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running packer uninstall command, error: " + err.Error())
		}
	}

	// Uninstalling Vagrant Parallels Plugin
	vagrantPresent := c.findPath("vagrant")
	if vagrantPresent != "" {
		cmd = c.findPath("vagrant")
		arguments := []string{"plugin", "uninstall", "vagrant-parallels"}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running vagrant uninstall plugin command, error: " + err.Error())
		}

		// Uninstalling Vagrant
		cmd = c.findPath("brew")
		arguments = []string{"uninstall", "hashicorp-vagrant"}
		_, err = c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running vagrant uninstall command, error: " + err.Error())
		}
	}

	return nil
}

func (c *ParallelsServerClient) InstallParallelsDesktop() error {
	// checking if is already installed
	cmd := c.findPath("prlctl")
	arguments := []string{"--version"}
	_, err := c.client.RunCommand(cmd, arguments)
	if err == nil {
		return nil
	}

	// Installing parallels desktop using command line
	cmd = c.findPath("brew")
	arguments = []string{"install", "parallels"}
	_, err = c.client.RunCommand(cmd, arguments)
	if err != nil {
		return errors.New("Error running parallels install command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) UninstallParallelsDesktop() error {
	// checking if the prlctl is indeed installed, if not we do not need to do anything
	cmd := c.findPath("prlctl")
	arguments := []string{"--version"}
	_, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return err
	}

	cmd = c.findPath("brew")
	arguments = []string{"uninstall", "parallels"}
	_, err = c.client.RunCommand(cmd, arguments)
	if err != nil {
		return errors.New("Error running parallels uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *ParallelsServerClient) GetLicense() (*ParallelsDesktopLicense, error) {
	cmd := c.findPath("prlsrvctl")
	arguments := []string{"info", "--json"}
	output, err := c.client.RunCommand(cmd, arguments)
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

	parallelsLicense := ParallelsDesktopLicense{}
	parallelsLicense.FromClientModel(parallelsInfo.License)
	return &parallelsLicense, nil
}

func (c *ParallelsServerClient) InstallLicense(key string, username, password string) error {
	if username != "" && password != "" {
		// Generating the license password file
		cmd := "echo"
		arguments := []string{password, ">", "~/parallels_password.txt"}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return err
		}

		cmd = c.findPath("prlsrvctl")
		arguments = []string{"web-portal", "signin", username, "--read-passwd", "~/parallels_password.txt"}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return err
		}
	}

	cmd := c.findPath("prlsrvctl")
	arguments := []string{"install-license", "--key", key, "--activate-online-immediately"}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	return nil
}

func (c *ParallelsServerClient) DeactivateLicense() error {
	cmd := c.findPath("prlsrvctl")
	arguments := []string{"deactivate-license", "--skip-network-errors"}

	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
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

func (c *ParallelsServerClient) InstallApiService(license string, config ParallelsDesktopApiConfig) (string, error) {
	var releaseDetails clientmodels.GithubRelease
	var baseUrl string

	if config.InstallVersion.ValueString() == "" || config.InstallVersion.ValueString() == "latest" {
		tflog.Info(c.ctx, "Parallels Desktop DevOps Service version not specified, installing latest version")
		baseUrl = "https://api.github.com/repos/Parallels/prl-devops-service/releases/latest"
	} else {
		tflog.Info(c.ctx, "Parallels Desktop DevOps Service version specified, installing version: "+config.InstallVersion.ValueString())
		baseUrl = "https://api.github.com/repos/Parallels/prl-devops-service/releases/tags/" + config.InstallVersion.ValueString()
	}

	caller := helpers.NewHttpCaller(c.ctx)
	if _, err := caller.GetDataFromClient(baseUrl, nil, nil, &releaseDetails); err != nil {
		return "", err
	}

	finalVersion := strings.ReplaceAll(releaseDetails.TagName, "v", "")
	tflog.Info(c.ctx, "Parallels Desktop DevOps Service Latest version: "+releaseDetails.TagName)

	tflog.Info(c.ctx, "Getting the url for the correct asset to download")

	// Getting the right asset
	var assetUrl string
	var os string
	var arch string
	_, ok := c.client.(*ssh.SshClient)
	if !ok {
		os = runtime.GOOS
		arch = runtime.GOARCH
	} else {
		systemOs, err := c.client.RunCommand("uname", []string{"-s"})
		if err != nil {
			os = "darwin"
		} else {
			os = strings.ReplaceAll(strings.ToLower(systemOs), "\n", "")
		}

		systemArchitecture, err := c.client.RunCommand("uname", []string{"-m"})
		systemArchitecture = strings.ReplaceAll(systemArchitecture, "\n", "")
		if err != nil {
			arch = "arm64"
		} else {
			if systemArchitecture != "arm64" {
				arch = "amd64"
			} else {
				arch = systemArchitecture
			}
		}
	}
	assetSuffix := fmt.Sprintf("%s-%s", os, arch)

	for _, asset := range releaseDetails.Assets {
		if strings.Contains(asset.Name, assetSuffix) {
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
	cmd := c.findPath("curl")
	arguments := []string{"-L", "-o", "/tmp/prl-devops-service.tar.gz", assetUrl}
	_, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Extracting the asset")
	// Extracting the asset
	cmd = "sudo"
	arguments = []string{c.findPath("tar"), "-xzf", "/tmp/prl-devops-service.tar.gz", "-C", installPath}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return "", err
	}

	cmd = "sudo"
	arguments = []string{"rm", "-f", "/tmp/prl-devops-service.tar.gz"}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Installing the service")
	if os == "darwin" {
		tflog.Info(c.ctx, "Generating installation config file")
		configPath := "/tmp/config.json"
		err = c.generateConfigFile(configPath, config)
		if err != nil {
			return "", err
		}

		tflog.Info(c.ctx, "Installing service in the launchd daemon")
		cmd = "sudo"

		arguments = []string{installPath + "/prldevops", "install", "--file=" + configPath}
		out, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			tflog.Info(c.ctx, "Error installing service: \n"+out)
			return "", err
		}

		tflog.Info(c.ctx, "Cleaning configuration")
		// cmd = "sudo"
		// arguments = []string{"rm", "-f", configPath}
		// _, err = c.client.RunCommand(cmd, arguments)
		// if err != nil {
		// 	return "", err
		// }
	} else {
		tflog.Error(c.ctx, "Unsupported OS: "+os)
	}

	tflog.Info(c.ctx, "Done")
	return finalVersion, nil
}

func (c *ParallelsServerClient) UninstallApiService() error {
	tflog.Info(c.ctx, "Uninstalling the Parallels Desktop DevOps Service")
	executableName, err := c.getExecutableName(installPath)
	if err != nil {
		return err
	}

	cmd := "sudo"
	executablePath := filepath.Join(installPath, executableName)
	arguments := []string{executablePath, "uninstall"}
	if executableName == "pd-api-service" {
		arguments = []string{executablePath, "uninstall"}
	} else {
		arguments = append(arguments, "--full")
	}
	uninstallOut, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		tflog.Error(c.ctx, "Error uninstalling service: \n"+uninstallOut)
		return err
	}

	cmd = "sudo"
	arguments = []string{"rm", "-f", executablePath}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	cmd = "sudo"
	arguments = []string{"rm", "-f", "/tmp/api-service.job.*"}
	if _, err = c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	// Removing database directory
	if c.fileExists("/etc/prl-devops-service") {
		cmd = "sudo"
		arguments = []string{"rm", "-rf", "/etc/prl-devops-service"}
		if _, err = c.client.RunCommand(cmd, arguments); err != nil {
			return err
		}
	}

	return nil
}

func (c *ParallelsServerClient) GetApiVersion() (string, error) {
	executableName, err := c.getExecutableName(installPath)
	if err != nil {
		return "", err
	}

	executablePath := filepath.Join(installPath, executableName)
	cmd := executablePath
	arguments := []string{"version"}
	if executableName == "pd-api-service" {
		arguments = []string{"--version"}
	}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *ParallelsServerClient) GetPackerVersion() (string, error) {
	cmd := c.findPath("packer")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *ParallelsServerClient) GetVagrantVersion() (string, error) {
	cmd := c.findPath("vagrant")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "Vagrant  ", ""), nil
}

func (c *ParallelsServerClient) GetGitVersion() (string, error) {
	cmd := c.findPath("git")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "git version ", ""), nil
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

func (c *ParallelsServerClient) generateConfigFile(path string, config ParallelsDesktopApiConfig) error {
	configJson := make(map[string]interface{})
	if config.Port.ValueString() != "" {
		configJson["port"] = config.Port.ValueString()
	}
	if config.Prefix.ValueString() != "" {
		configJson["prefix"] = config.Prefix.ValueString()
	}
	if config.RootPassword.ValueString() != "" {
		configJson["root_password"] = config.RootPassword.ValueString()
	}
	if config.HmacSecret.ValueString() != "" {
		configJson["hmac_secret"] = config.HmacSecret.ValueString()
	}
	if config.EncryptionRsaKey.ValueString() != "" {
		configJson["encryption_rsa_key"] = config.EncryptionRsaKey.ValueString()
	}
	if config.LogLevel.ValueString() != "" {
		configJson["log_level"] = config.LogLevel.ValueString()
	}
	if config.EnableTLS.ValueBool() {
		configJson["enable_tls"] = true
	}
	if config.TLSPort.ValueString() != "" {
		configJson["tls_port"] = config.TLSPort.ValueString()
	}
	if config.TLSCertificate.ValueString() != "" {
		configJson["tls_certificate"] = config.TLSCertificate.ValueString()
	}
	if config.TLSPrivateKey.ValueString() != "" {
		configJson["tls_private_key"] = config.TLSPrivateKey.ValueString()
	}
	if config.DisableCatalogCaching.ValueBool() {
		configJson["disable_catalog_caching"] = true
	}
	if config.TokenDurationMinutes.ValueString() != "" {
		configJson["token_duration_minutes"] = config.TokenDurationMinutes.ValueString()
	}
	if config.Mode.ValueString() != "" {
		configJson["mode"] = config.Mode.ValueString()
	}
	if config.UseOrchestratorResources.ValueBool() {
		configJson["use_orchestrator_resources"] = true
	}

	confJson, err := json.Marshal(configJson)
	if err != nil {
		return err
	}

	cmd := "touch"
	arguments := []string{path}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	cmd = "echo"
	arguments = []string{"'" + string(confJson) + "' ", ">", path}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	return nil
}

func (s *ParallelsServerClient) findPath(cmd string) string {
	tflog.Info(s.ctx, "Getting "+cmd+" executable")
	out, err := s.client.RunCommand("which", []string{cmd})
	path := strings.ReplaceAll(strings.TrimSpace(out), "\n", "")
	if err != nil || path == "" {
		tflog.Info(s.ctx, cmd+" executable not found, trying to find it in the default locations")
		path = ""
	}

	folders := []string{"/usr/local/bin", "/usr/bin", "/bin", "/usr/sbin", "/sbin", "/opt/homebrew/bin"}

	for _, folder := range folders {
		if _, err := s.client.RunCommand("ls", []string{filepath.Join(folder, cmd)}); err == nil {
			path = filepath.Join(folder, cmd)
			break
		}
	}

	return path
}

func (c *ParallelsServerClient) getExecutableName(installPath string) (string, error) {
	executableName := ""
	for _, exec := range executableNames {
		execPath := filepath.Join(installPath, exec)
		if c.fileExists(execPath) {
			executableName = exec
			break
		}
	}

	if executableName == "" {
		return "", errors.New("Parallels Desktop DevOps Service not found")
	}

	return executableName, nil
}

func (c *ParallelsServerClient) fileExists(filepath string) bool {
	cmd := "ls"
	arguments := []string{filepath}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return false
	}

	return true
}
