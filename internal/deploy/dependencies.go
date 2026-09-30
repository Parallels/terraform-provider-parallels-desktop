package deploy

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"terraform-provider-parallels-desktop/internal/localclient"
)

// Markers distinguish an explicit absence from transport/probe failures or banners.
const discoveryScript = `set -eu
tool=$1
candidate=$(command -v "$tool" 2>/dev/null || true)
case "$candidate" in
 /*)
  if [ -f "$candidate" ] && [ -x "$candidate" ]; then
   printf '\nPRL_TOOL_FOUND:%s\n' "$candidate"
   exit 0
  fi
  ;;
esac
for folder in /opt/homebrew/bin /usr/local/bin /usr/bin /bin /usr/sbin /sbin "$HOME/bin"; do
 candidate="$folder/$tool"
 if [ -f "$candidate" ] && [ -x "$candidate" ]; then
  printf '\nPRL_TOOL_FOUND:%s\n' "$candidate"
  exit 0
 fi
done
printf '\nPRL_TOOL_ABSENT\n'
`

func (c *DevOpsServiceClient) findPath(ctx context.Context, tool string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if path := c.paths[tool]; path != "" {
		return path, true, nil
	}
	output, err := c.client.RunCommandContext(ctx, "/bin/sh", []string{"-c", discoveryScript, "discover-tool", tool})
	if err != nil {
		return "", false, fmt.Errorf("discover-tool %s: %w", tool, err)
	}
	var path string
	results := 0
	foundMarker := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "PRL_TOOL_ABSENT" {
			results++
		}
		if strings.HasPrefix(line, "PRL_TOOL_FOUND:") {
			results++
			foundMarker = true
			path = strings.TrimPrefix(line, "PRL_TOOL_FOUND:")
		}
	}
	if results != 1 || (foundMarker && path == "") || (path != "" && (!filepath.IsAbs(path) || filepath.Base(path) != tool)) {
		return "", false, fmt.Errorf("discover-tool %s: invalid probe response", tool)
	}
	if path != "" {
		c.paths[tool] = path
	}
	return path, path != "", nil
}

type executableNotFoundError struct{ tool string }

func (e *executableNotFoundError) Error() string {
	return fmt.Sprintf("required executable %s not found", e.tool)
}

func executableMissing(err error, tool string) bool {
	var missing *executableNotFoundError
	return errors.As(err, &missing) && missing.tool == tool
}

func (c *DevOpsServiceClient) requirePath(ctx context.Context, tool string) (string, error) {
	path, found, err := c.findPath(ctx, tool)
	if err != nil {
		return "", err
	}
	if !found {
		return "", &executableNotFoundError{tool: tool}
	}
	return path, nil
}
func quoteShell(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }

func (c *DevOpsServiceClient) preflightSudo(ctx context.Context) error {
	if _, err := c.client.RunCommandContext(ctx, "sudo", []string{"-n", "true"}); err != nil {
		return fmt.Errorf("dependency privilege preflight: noninteractive sudo is required; configure deployment sudo rights before applying (the provider does not modify sudoers): %w", err)
	}
	return nil
}

func (c *DevOpsServiceClient) InstallDependencies(ctx context.Context, requested []string) ([]string, error) {
	installed := []string{}
	for _, dep := range requested {
		switch strings.ToLower(dep) {
		case "brew", "git", "packer", "vagrant":
		default:
			return installed, fmt.Errorf("unsupported dependency %q", dep)
		}
	}
	if err := c.preflightSudo(ctx); err != nil {
		return installed, err
	}
	_, hadBrew, err := c.findPath(ctx, "brew")
	if err != nil {
		return installed, err
	}
	if err = c.InstallBrew(ctx); err != nil {
		return installed, err
	}
	if !hadBrew {
		installed = append(installed, "brew")
	}
	for _, dep := range requested {
		dep = strings.ToLower(dep)
		_, found, err := c.findPath(ctx, dep)
		if err != nil {
			return installed, err
		}
		if found {
			continue
		}
		if err = c.installTool(ctx, dep); err != nil {
			return installed, err
		}
		installed = append(installed, dep)
		if dep == "vagrant" {
			if err = c.installVagrantPlugin(ctx); err != nil {
				return installed, err
			}
		}
	}
	return installed, nil
}

func (c *DevOpsServiceClient) InstallBrew(ctx context.Context) error {
	_, found, err := c.findPath(ctx, "brew")
	if err != nil || found {
		return err
	}
	if err = c.preflightSudo(ctx); err != nil {
		return err
	}
	script := `set -eu
export NONINTERACTIVE=1
installer=$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)
/bin/bash -c "$installer"
`
	if _, err = c.client.RunCommandContext(ctx, "/bin/bash", []string{"-c", script}); err != nil {
		return fmt.Errorf("install Homebrew: %w", err)
	}
	delete(c.paths, "brew")
	_, err = c.requirePath(ctx, "brew")
	return err
}
func (c *DevOpsServiceClient) UninstallBrew() error { return nil }
func (c *DevOpsServiceClient) installTool(ctx context.Context, tool string) error {
	brew, err := c.requirePath(ctx, "brew")
	if err != nil {
		return err
	}
	delete(c.paths, tool)
	if _, err = c.client.RunCommandContext(ctx, brew, []string{"install", tool}); err != nil {
		return fmt.Errorf("install %s: %w", tool, err)
	}
	delete(c.paths, tool)
	_, err = c.requirePath(ctx, tool)
	return err
}

func (c *DevOpsServiceClient) uninstallTool(ctx context.Context, tool string) error {
	_, found, err := c.findPath(ctx, tool)
	if err != nil || !found {
		return err
	}
	brew, err := c.requirePath(ctx, "brew")
	if err != nil {
		return err
	}
	defer delete(c.paths, tool)
	_, err = c.client.RunCommandContext(ctx, brew, []string{"uninstall", tool})
	return err
}
func (c *DevOpsServiceClient) InstallGit(ctx context.Context) error { return c.installTool(ctx, "git") }
func (c *DevOpsServiceClient) InstallPacker(ctx context.Context) error {
	return c.installTool(ctx, "packer")
}

func (c *DevOpsServiceClient) UninstallGit(ctx context.Context) error {
	return c.uninstallTool(ctx, "git")
}

func (c *DevOpsServiceClient) UninstallPacker(ctx context.Context) error {
	return c.uninstallTool(ctx, "packer")
}

func (c *DevOpsServiceClient) installVagrantPlugin(ctx context.Context) error {
	executable, err := c.requirePath(ctx, "vagrant")
	if err != nil {
		return err
	}
	_, err = c.client.RunCommandContext(ctx, executable, []string{"plugin", "install", "vagrant-parallels"})
	return err
}

func (c *DevOpsServiceClient) InstallVagrant(ctx context.Context) error {
	if err := c.installTool(ctx, "vagrant"); err != nil {
		return err
	}
	return c.installVagrantPlugin(ctx)
}

func (c *DevOpsServiceClient) UninstallVagrant(ctx context.Context) error {
	executable, found, err := c.findPath(ctx, "vagrant")
	if err != nil || !found {
		return err
	}
	if _, err = c.client.RunCommandContext(ctx, executable, []string{"plugin", "uninstall", "vagrant-parallels"}); err != nil {
		return err
	}
	return c.uninstallTool(ctx, "vagrant")
}

func (c *DevOpsServiceClient) UninstallDependencies(ctx context.Context, installed []string) []error {
	errs := []error{}
	if _, local := c.client.(*localclient.LocalClient); local {
		return errs
	}
	for _, dep := range installed {
		var err error
		switch dep {
		case "brew":
			continue
		case "git", "packer":
			err = c.uninstallTool(ctx, dep)
		case "vagrant":
			err = c.UninstallVagrant(ctx)
		default:
			err = fmt.Errorf("unsupported dependency %q", dep)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (c *DevOpsServiceClient) InstallParallelsDesktop(ctx context.Context) error {
	executable, found, err := c.findPath(ctx, "prlctl")
	if err != nil {
		return err
	}
	if found {
		_, err = c.client.RunCommandContext(ctx, executable, []string{"--version"})
		return err
	}
	brew, err := c.requirePath(ctx, "brew")
	if err != nil {
		return err
	}
	if _, err = c.client.RunCommandContext(ctx, brew, []string{"install", "parallels"}); err != nil {
		return err
	}
	delete(c.paths, "prlctl")
	delete(c.paths, "prlsrvctl")
	_, err = c.requirePath(ctx, "prlctl")
	return err
}

func (c *DevOpsServiceClient) UninstallParallelsDesktop(ctx context.Context) error {
	if _, local := c.client.(*localclient.LocalClient); local {
		return nil
	}
	_, found, err := c.findPath(ctx, "prlctl")
	if err != nil || !found {
		return err
	}
	brew, err := c.requirePath(ctx, "brew")
	if err != nil {
		return err
	}
	defer delete(c.paths, "prlctl")
	defer delete(c.paths, "prlsrvctl")
	_, err = c.client.RunCommandContext(ctx, brew, []string{"uninstall", "parallels"})
	return err
}
