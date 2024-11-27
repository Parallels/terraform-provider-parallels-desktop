package deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"terraform-provider-parallels-desktop/internal/clientmodels"
	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/interfaces"
	"terraform-provider-parallels-desktop/internal/localclient"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	installPath     = "/usr/local/bin"
	executableNames = []string{"prldevops", "prldevops"}
)

type DevOpsServiceClient struct {
	client interfaces.CommandClient
	ctx    context.Context
}

type DevOpsServiceConfigFile struct {
	EnvironmentVariables map[string]string `json:"environment" yaml:"environment"`
}

func NewDevOpsServiceClient(ctx context.Context, client interfaces.CommandClient) *DevOpsServiceClient {
	return &DevOpsServiceClient{
		client: client,
		ctx:    ctx,
	}
}

func (c *DevOpsServiceClient) GetInfo() (*clientmodels.ParallelsServerInfo, error) {
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

func (c *DevOpsServiceClient) GetVersion() (string, error) {
	parallelsInfo, err := c.GetInfo()
	if err != nil {
		return "", err
	}

	return parallelsInfo.Version, nil
}

func (c *DevOpsServiceClient) RestartServer() error {
	cmd := "/Applications/Parallels\\ Desktop.app/Contents/MacOS/Parallels\\ Service"
	arguments := []string{"start"}
	_, _ = c.client.RunCommand(cmd, arguments)

	_, _ = c.client.RunCommand(cmd, arguments)

	return nil
}

func (c *DevOpsServiceClient) InstallDependencies(listToInstall []string) ([]string, error) {
	installed_dependencies := []string{}
	_, ok := c.client.(*localclient.LocalClient)

	if err := c.InstallBrew(); err != nil {
		return installed_dependencies, err
	}
	installed_dependencies = append(installed_dependencies, "brew")

	if !ok {
		for _, dep := range listToInstall {
			switch strings.ToLower(dep) {
			case "brew":
				brewPresent := c.findPath("brew")
				if brewPresent == "" {
					if err := c.InstallBrew(); err != nil {
						return installed_dependencies, err
					}
					isAlreadyInInstalledDependencies := false
					for _, installedDep := range installed_dependencies {
						if installedDep == "brew" {
							isAlreadyInInstalledDependencies = true
						}
					}
					if !isAlreadyInInstalledDependencies {
						installed_dependencies = append(installed_dependencies, "brew")
					}
				}
				// setting up sudo access for brew without password

				cmd := "echo"
				// " %v | sudo -S echo hello | sudo grep -q '^%v ALL=(ALL) NOPASSWD:ALL$' /etc/sudoers || echo '%v ALL=(ALL) NOPASSWD:ALL' | sudo tee -a /etc/sudoers", c.client.Password(), c.client.Username(), c.client.Username()
				sudoArgs := []string{
					c.client.Password(),
					"|",
					"sudo",
					"-S",
					"echo",
					"hello",
					"|",
					"sudo",
					"grep",
					"-q",
					fmt.Sprintf("'^%v ALL=(ALL) NOPASSWD:ALL$'", c.client.Username()),
					"/etc/sudoers",
					"||",
					"echo",
					fmt.Sprintf("'%v ALL=(ALL) NOPASSWD:ALL'", c.client.Username()),
					"|",
					"sudo",
					"tee",
					"-a",
					"/etc/sudoers",
				}

				_, err := c.client.RunCommand(cmd, sudoArgs)
				fullCommand := fmt.Sprintf("%v %v", cmd, strings.Join(sudoArgs, " "))
				tflog.Info(c.ctx, "Full command: "+fullCommand)
				if err != nil {
					return installed_dependencies, errors.New("Error setting up sudo access for brew without password, error: " + err.Error())
				}
			case "git":
				gitPresent := c.findPath("git")
				brewPresent := c.findPath("brew")
				if gitPresent == "" && brewPresent == "" {
					if err := c.InstallGit(); err != nil {
						return installed_dependencies, err
					}
					installed_dependencies = append(installed_dependencies, "git")
				}
			case "packer":
				packerPresent := c.findPath("packer")
				brewPresent := c.findPath("brew")
				if packerPresent == "" && brewPresent == "" {
					if err := c.InstallPacker(); err != nil {
						return installed_dependencies, err
					}
					installed_dependencies = append(installed_dependencies, "packer")
				}
			case "vagrant":
				vagrantPresent := c.findPath("vagrant")
				brewPresent := c.findPath("brew")
				if vagrantPresent == "" && brewPresent == "" {
					if err := c.InstallVagrant(); err != nil {
						return installed_dependencies, err
					}
					installed_dependencies = append(installed_dependencies, "vagrant")
				}
			default:
				return installed_dependencies, errors.New("Unsupported dependency " + dep + " to install")
			}
		}
	} else {
		return installed_dependencies, errors.New("Unsupported client")
	}

	return installed_dependencies, nil
}

func (c *DevOpsServiceClient) UninstallDependencies(installedDependencies []string) []error {
	_, ok := c.client.(*localclient.LocalClient)
	uninstalllErrors := []error{}
	if !ok {
		for _, dep := range installedDependencies {
			switch dep {
			case "brew":
				continue
			case "git":
				if err := c.UninstallGit(); err != nil {
					uninstalllErrors = append(uninstalllErrors, err)
				}
			case "packer":
				if err := c.UninstallPacker(); err != nil {
					uninstalllErrors = append(uninstalllErrors, err)
				}
			case "vagrant":
				if err := c.UninstallVagrant(); err != nil {
					uninstalllErrors = append(uninstalllErrors, err)
				}
			default:
				uninstalllErrors = append(uninstalllErrors, errors.New("Unsupported dependency"+dep+" to uninstall"))
			}
		}
	} else {
		uninstalllErrors = append(uninstalllErrors, errors.New("Unsupported client"))
	}

	return uninstalllErrors
}

func (c *DevOpsServiceClient) InstallBrew() error {
	// Installing Brew
	var cmd string
	var brewPath string
	var arguments []string

	brewPath = c.findPath("brew")
	if brewPath == "" {
		cmd = "/bin/bash"
		arguments = []string{"-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return errors.New("Error running brew install command, error: " + err.Error())
		}
	}

	if c.findPath("brew") == "" {
		return errors.New("Error running brew install command, error: brew not found")
	}

	return nil
}

func (c *DevOpsServiceClient) UninstallBrew() error {
	// we do not uninstall brew, as it is a system package manager
	return nil
}

func (c *DevOpsServiceClient) InstallGit() error {
	// Installing Git
	cmd := c.findPath("brew")
	if cmd == "" {
		return errors.New("Error running git install command, error: brew not found")
	}

	arguments := []string{"install", "git"}
	out, err := c.client.RunCommand(cmd, arguments)
	tflog.Info(c.ctx, "Git install output: "+out)
	if err != nil {
		return errors.New("Error running git install command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) UninstallGit() error {
	// Uninstalling Git
	gitPresent := c.findPath("git")
	if gitPresent == "" {
		return nil
	}

	cmd := c.findPath("brew")
	if cmd == "" {
		return errors.New("Error running git install command, error: brew not found")
	}

	arguments := []string{"uninstall", "git"}
	out, err := c.client.RunCommand(cmd, arguments)
	tflog.Info(c.ctx, "Git uninstall output: "+out)
	if err != nil {
		return errors.New("Error running git uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) InstallPacker() error {
	// Installing Packer
	cmd := c.findPath("brew")
	if c.findPath("brew") == "" {
		return nil
	}

	arguments := []string{"install", "packer"}
	out, err := c.client.RunCommand(cmd, arguments)
	tflog.Info(c.ctx, "Packer install output: "+out)
	if err != nil {
		return errors.New("Error running packer install command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) UninstallPacker() error {
	// Uninstalling Packer
	packerPresent := c.findPath("packer")
	if packerPresent == "" {
		return nil
	}

	cmd := c.findPath("brew")
	if cmd == "" {
		return errors.New("Error running packer uninstall command, error: brew not found")
	}

	arguments := []string{"uninstall", "packer"}
	out, err := c.client.RunCommand(cmd, arguments)
	tflog.Info(c.ctx, "Packer uninstall output: "+out)
	if err != nil {
		return errors.New("Error running packer uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) InstallVagrant() error {
	// Installing Vagrant
	cmd := c.findPath("brew")

	if c.findPath("brew") == "" {
		return nil
	}

	arguments := []string{"install", "vagrant"}
	out, err := c.client.RunCommand(cmd, arguments)
	tflog.Info(c.ctx, "Vagrant install output: "+out)
	if err != nil {
		return errors.New("Error running vagrant install command, error: " + err.Error())
	}

	vagrantCommand := c.findPath("vagrant")
	if vagrantCommand == "" {
		return errors.New("Error running vagrant install command, error: vagrant not found")
	}

	// Installing Vagrant Parallels Plugin
	arguments = []string{"plugin", "install", "vagrant-parallels"}
	out, err = c.client.RunCommand(vagrantCommand, arguments)
	tflog.Info(c.ctx, "Vagrant plugin install output: "+out)
	if err != nil {
		return errors.New("Error running vagrant plugin install command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) UninstallVagrant() error {
	// Uninstalling Vagrant Parallels Plugin
	vagrantPresent := c.findPath("vagrant")
	if vagrantPresent == "" {
		return nil
	}
	brewCmd := c.findPath("brew")
	if brewCmd == "" {
		return errors.New("Error running vagrant uninstall plugin command, error: brew not found")
	}

	vagrantCmd := c.findPath("vagrant")
	arguments := []string{"plugin", "uninstall", "vagrant-parallels"}
	_, err := c.client.RunCommand(vagrantCmd, arguments)
	if err != nil {
		return errors.New("Error running vagrant uninstall plugin command, error: " + err.Error())
	}

	// Uninstalling Vagrant
	arguments = []string{"uninstall", "vagrant"}
	out, err := c.client.RunCommand(brewCmd, arguments)
	tflog.Info(c.ctx, "Vagrant uninstall output: "+out)
	if err != nil {
		return errors.New("Error running vagrant uninstall command, error: " + err.Error())
	}

	return nil
}

func (c *DevOpsServiceClient) InstallParallelsDesktop() error {
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

func (c *DevOpsServiceClient) UninstallParallelsDesktop() error {
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

func (c *DevOpsServiceClient) GetLicense() (*models.ParallelsDesktopLicense, error) {
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

	parallelsLicense := models.ParallelsDesktopLicense{}
	parallelsLicense.FromClientModel(parallelsInfo.License)
	return &parallelsLicense, nil
}

func (c *DevOpsServiceClient) InstallLicense(key string, username, password string) error {
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

func (c *DevOpsServiceClient) DeactivateLicense() error {
	cmd := c.findPath("prlsrvctl")
	arguments := []string{"deactivate-license", "--skip-network-errors"}

	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return err
	}

	return nil
}

func (c *DevOpsServiceClient) CompareLicenses(license string) (bool, error) {
	currentLicense, err := c.GetLicense()
	if err != nil || currentLicense == nil {
		return false, err
	}

	if currentLicense.Key.IsUnknown() || currentLicense.Key.IsNull() {
		tflog.Info(c.ctx, "Current license: "+currentLicense.Key.ValueString())
	} else {
		tflog.Info(c.ctx, "Current license key is nil")
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

func (c *DevOpsServiceClient) InstallDevOpsService(license string, config models.ParallelsDesktopDevopsConfigV2) (string, error) {
	// Installing DevOps Service

	devopsPath := c.findPath("prldevops")
	if devopsPath == "" {
		cmd := "/bin/bash"
		arguments := []string{"-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Parallels/prl-devops-service/main/scripts/install.sh)\"", "-", "--no-service"}
		if config.DevOpsVersion.ValueString() != "" && config.DevOpsVersion.ValueString() != "latest" && !config.UseLatestBeta.ValueBool() {
			arguments = append(arguments, "--version", config.DevOpsVersion.ValueString())
		}
		if config.UseLatestBeta.ValueBool() {
			arguments = append(arguments, "--pre-release")
		}
		_, err := c.client.RunCommand(cmd, arguments)
		if err != nil {
			return "", errors.New("Error running devops install command, error: " + err.Error())
		}
	}

	devopsPath = c.findPath("prldevops")
	if devopsPath == "" {
		return "", errors.New("Error running devops install command, error: brew not found")
	}

	folderPath := c.findPathFolder("prldevops")
	if folderPath == "" {
		return "", errors.New("Error running devops install command, error: prldevops folder not found")
	}

	// Setting the environment variables
	if config.EnvironmentVariables != nil {
		configFile := DevOpsServiceConfigFile{
			EnvironmentVariables: make(map[string]string),
		}

		for key, envVar := range config.EnvironmentVariables {
			configFile.EnvironmentVariables[key] = envVar.ValueString()
		}

		// Setting the environment variables for the prldevops service port forwarding
		if config.EnablePortForwarding.ValueBool() {
			configFile.EnvironmentVariables["ENABLE_REVERSE_PROXY"] = "true"
		}

		yamlConfig, err := yaml.Marshal(configFile)
		if err != nil {
			return "", err
		}

		configFilePath := filepath.Join("/tmp", "config.yaml")
		cmd := "echo"
		arguments := []string{"'" + string(yamlConfig) + "' ", ">", configFilePath}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return "", err
		}

		cmd = "sudo"
		arguments = []string{"cp", configFilePath, folderPath}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return "", err
		}

		cmd = "sudo"
		arguments = []string{"chown", "root:wheel", filepath.Join(folderPath, "config.yaml")}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return "", err
		}

		cmd = "sudo"
		arguments = []string{"chmod", "644", filepath.Join(folderPath, "config.yaml")}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return "", err
		}

		cmd = "rm"
		arguments = []string{configFilePath}
		if _, err := c.client.RunCommand(cmd, arguments); err != nil {
			return "", err
		}
	}

	configPath, err := c.generateConfigFile(config)
	if err != nil {
		return "", err
	}

	installServiceCmd := "sudo"
	installServiceArgs := []string{devopsPath, "install", "service", "--file=" + configPath}
	_, err = c.client.RunCommand(installServiceCmd, installServiceArgs)
	if err != nil {
		return "", err
	}

	removeConfigCmd := "rm"
	removeConfigArgs := []string{configPath}
	_, err = c.client.RunCommand(removeConfigCmd, removeConfigArgs)
	if err != nil {
		return "", err
	}

	finalVersion, err := c.GetDevOpsVersion()
	if err != nil {
		return "", err
	}

	tflog.Info(c.ctx, "Done")
	return finalVersion, nil
}

func (c *DevOpsServiceClient) UninstallDevOpsService() error {
	tflog.Info(c.ctx, "Uninstalling the Parallels Desktop DevOps Service")

	devopsPath := c.findPath("prldevops")
	if devopsPath == "" {
		return nil
	}

	cmd := "/bin/bash"
	arguments := []string{"-c", "\"$(curl -fsSL https://raw.githubusercontent.com/Parallels/prl-devops-service/main/scripts/install.sh)\"", "--", "--uninstall"}

	_, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return err
	}

	return nil
}

func (c *DevOpsServiceClient) GetDevOpsVersion() (string, error) {
	executableName, err := c.getExecutableName(installPath)
	if err != nil {
		return "", err
	}

	executablePath := filepath.Join(installPath, executableName)
	cmd := executablePath
	arguments := []string{"version"}
	if executableName == "prldevops" {
		arguments = []string{"--version"}
	}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *DevOpsServiceClient) GetPackerVersion() (string, error) {
	cmd := c.findPath("packer")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(output, "\n", ""), nil
}

func (c *DevOpsServiceClient) GetVagrantVersion() (string, error) {
	cmd := c.findPath("vagrant")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "Vagrant  ", ""), nil
}

func (c *DevOpsServiceClient) GetGitVersion() (string, error) {
	cmd := c.findPath("git")
	arguments := []string{"--version"}
	output, err := c.client.RunCommand(cmd, arguments)
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(strings.ReplaceAll(output, "\n", ""), "git version ", ""), nil
}

func (c *DevOpsServiceClient) GenerateDefaultRootPassword() (string, error) {
	info, err := c.GetInfo()
	if err != nil {
		return "", err
	}

	key := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(info.License.Key, "-", ""), "*", ""))
	hid := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(info.HardwareID, "-", ""), "{", ""), "}", ""))

	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", key, hid)))

	return encoded, nil
}

func (c *DevOpsServiceClient) generateConfigFile(config models.ParallelsDesktopDevopsConfigV2) (string, error) {
	configPath := "/tmp/service_config.json"
	configMap := make(map[string]interface{})
	if config.Port.ValueString() != "" {
		configMap["port"] = config.Port.ValueString()
	}
	if config.Prefix.ValueString() != "" {
		configMap["prefix"] = config.Prefix.ValueString()
	}
	if config.RootPassword.ValueString() != "" {
		configMap["root_password"] = config.RootPassword.ValueString()
	}
	if config.HmacSecret.ValueString() != "" {
		configMap["hmac_secret"] = config.HmacSecret.ValueString()
	}
	if config.EncryptionRsaKey.ValueString() != "" {
		configMap["encryption_rsa_key"] = config.EncryptionRsaKey.ValueString()
	}
	if config.LogLevel.ValueString() != "" {
		configMap["log_level"] = config.LogLevel.ValueString()
	}
	if config.EnableTLS.ValueBool() {
		configMap["enable_tls"] = true
	}
	if config.TLSPort.ValueString() != "" {
		configMap["tls_port"] = config.TLSPort.ValueString()
	}
	if config.TLSCertificate.ValueString() != "" {
		configMap["tls_certificate"] = config.TLSCertificate.ValueString()
	}
	if config.TLSPrivateKey.ValueString() != "" {
		configMap["tls_private_key"] = config.TLSPrivateKey.ValueString()
	}
	if config.DisableCatalogCaching.ValueBool() {
		configMap["disable_catalog_caching"] = true
	}
	if config.TokenDurationMinutes.ValueString() != "" {
		configMap["token_duration_minutes"] = config.TokenDurationMinutes.ValueString()
	}
	if config.Mode.ValueString() != "" {
		configMap["mode"] = config.Mode.ValueString()
	}
	if config.UseOrchestratorResources.ValueBool() {
		configMap["use_orchestrator_resources"] = true
	}
	if config.SystemReservedMemory.ValueString() != "" {
		configMap["system_reserved_memory"] = config.SystemReservedMemory.ValueString()
	}
	if config.SystemReservedCpu.ValueString() != "" {
		configMap["system_reserved_cpu"] = config.SystemReservedCpu.ValueString()
	}
	if config.SystemReservedDisk.ValueString() != "" {
		configMap["system_reserved_disk"] = config.SystemReservedDisk.ValueString()
	}
	if config.EnableLogging.ValueBool() {
		configMap["log_output"] = true
	}

	jsonConfig, err := json.Marshal(configMap)
	if err != nil {
		return "", err
	}

	cmd := "echo"
	arguments := []string{"'" + string(jsonConfig) + "' ", ">", configPath}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return "", err
	}

	return configPath, nil
}

func (s *DevOpsServiceClient) findPath(cmd string) string {
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
			tflog.Info(s.ctx, "Found "+cmd+" executable at "+path)
			break
		}
	}

	return path
}

func (s *DevOpsServiceClient) findPathFolder(cmd string) string {
	tflog.Info(s.ctx, "Getting "+cmd+" executable folder")
	path := s.findPath(cmd)
	if path == "" {
		return ""
	}
	folder := filepath.Dir(path)
	tflog.Info(s.ctx, "Found "+cmd+" executable folder at "+folder)
	return folder
}

func (c *DevOpsServiceClient) getExecutableName(installPath string) (string, error) {
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

func (c *DevOpsServiceClient) fileExists(filepath string) bool {
	cmd := "ls"
	arguments := []string{filepath}
	if _, err := c.client.RunCommand(cmd, arguments); err != nil {
		return false
	}

	return true
}
