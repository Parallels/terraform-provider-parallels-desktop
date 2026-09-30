package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"terraform-provider-parallels-desktop/internal/deploy/models"
	"terraform-provider-parallels-desktop/internal/localclient"
)

type recordedCommand struct {
	command string
	args    []string
	input   string
}
type fakeCommands struct {
	tools        map[string]string
	probes       map[string]int
	records      []recordedCommand
	probeError   error
	failTool     string
	sudoError    error
	installError error
	pluginError  error
	versionError error
	output       string
	noInstall    bool
}

func newFakeCommands() *fakeCommands {
	return &fakeCommands{tools: map[string]string{}, probes: map[string]int{}}
}
func (f *fakeCommands) Username() string { return "deploy" }
func (f *fakeCommands) Password() string { return "secret'\"$`" }
func (f *fakeCommands) Close() error     { return nil }
func (f *fakeCommands) RunCommand(cmd string, args []string) (string, error) {
	return f.RunCommandContext(context.Background(), cmd, args)
}

func (f *fakeCommands) RunCommandContext(ctx context.Context, cmd string, args []string) (string, error) {
	return f.RunCommandInput(ctx, cmd, args, nil)
}

func (f *fakeCommands) RunCommandInput(ctx context.Context, cmd string, args []string, input io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	record := recordedCommand{command: cmd, args: append([]string(nil), args...)}
	if input != nil {
		bytes, _ := io.ReadAll(input)
		record.input = string(bytes)
	}
	f.records = append(f.records, record)
	if cmd == "sudo" {
		return "", f.sudoError
	}
	if cmd == "/bin/sh" && len(args) == 4 && args[2] == "discover-tool" {
		tool := args[3]
		f.probes[tool]++
		if f.probeError != nil && (f.failTool == "" || f.failTool == tool) {
			return "", f.probeError
		}
		if f.output != "" {
			return f.output, nil
		}
		if path := f.tools[tool]; path != "" {
			return "Welcome to the host\nPRL_TOOL_FOUND:" + path + "\n", nil
		}
		return "PRL_TOOL_ABSENT\n", nil
	}
	if cmd == "/bin/bash" && strings.Contains(strings.Join(args, " "), "Homebrew/install") {
		if f.installError != nil {
			return "", f.installError
		}
		if !f.noInstall {
			f.tools["brew"] = "/opt/homebrew/bin/brew"
		}
		return "", nil
	}
	if filepath.Base(cmd) == "brew" && len(args) == 2 {
		if f.installError != nil {
			return "", f.installError
		}
		if args[0] == "install" && !f.noInstall {
			f.tools[args[1]] = "/opt/homebrew/bin/" + args[1]
		}
		if args[0] == "uninstall" {
			delete(f.tools, args[1])
		}
		return "", nil
	}
	if filepath.Base(cmd) == "vagrant" && len(args) > 0 && args[0] == "plugin" {
		return "", f.pluginError
	}
	if len(args) > 0 && args[0] == "--version" {
		return "", f.versionError
	}
	return "", nil
}

func TestDependencyDiscoveryFailureDoesNotInstall(t *testing.T) {
	failure := errors.New("handshake reset")
	f := newFakeCommands()
	f.probeError = failure
	c := NewDevOpsServiceClient(context.Background(), f)
	installed, err := c.InstallDependencies(context.Background(), []string{"brew", "git"})
	if !errors.Is(err, failure) || len(installed) != 0 {
		t.Fatalf("installed=%v error=%v", installed, err)
	}
	for _, cmd := range f.records {
		if cmd.command != "sudo" && cmd.command != "/bin/sh" {
			t.Fatalf("unexpected mutation: %v", cmd)
		}
	}
	f.probeError = nil
	f.tools["brew"] = "/opt/homebrew/bin/brew"
	path, found, err := c.findPath(context.Background(), "brew")
	if err != nil || !found || path != f.tools["brew"] || f.probes["brew"] != 2 {
		t.Fatalf("failure cached as absence: %s %v", path, err)
	}
}

func TestInstallMissingDependenciesAndTrackOwnership(t *testing.T) {
	f := newFakeCommands()
	f.tools["brew"] = "/opt/homebrew/bin/brew"
	f.tools["git"] = "/usr/bin/git"
	c := NewDevOpsServiceClient(context.Background(), f)
	installed, err := c.InstallDependencies(context.Background(), []string{"brew", "git", "vagrant", "packer", "vagrant"})
	if err != nil || !reflect.DeepEqual(installed, []string{"vagrant", "packer"}) {
		t.Fatalf("%v %v", installed, err)
	}
	if f.probes["brew"] != 1 || f.probes["git"] != 1 {
		t.Fatal("successful paths not cached")
	}
	if f.probes["vagrant"] != 2 || f.probes["packer"] != 2 {
		t.Fatalf("tools not verified after install: %v", f.probes)
	}
	if errs := c.UninstallDependencies(context.Background(), installed); len(errs) > 0 {
		t.Fatal(errs)
	}
	if f.tools["brew"] == "" || f.tools["git"] == "" {
		t.Fatal("pre-existing tools removed")
	}
	if f.tools["vagrant"] != "" || f.tools["packer"] != "" {
		t.Fatal("installed tools not removed")
	}
	_, found, err := c.findPath(context.Background(), "vagrant")
	if err != nil || found {
		t.Fatal("uninstall did not invalidate cache")
	}
	for _, record := range f.records {
		if strings.Contains(strings.Join(record.args, " "), "sudoers") {
			t.Fatal("sudoers mutated")
		}
	}
}

func TestInstallBrewNoninteractiveAndVerify(t *testing.T) {
	f := newFakeCommands()
	c := NewDevOpsServiceClient(context.Background(), f)
	installed, err := c.InstallDependencies(context.Background(), []string{"brew"})
	if err != nil || !reflect.DeepEqual(installed, []string{"brew"}) {
		t.Fatalf("%v %v", installed, err)
	}
	scripts := 0
	for _, record := range f.records {
		if record.command == "/bin/bash" {
			scripts++
			if !strings.Contains(record.args[1], "NONINTERACTIVE=1") {
				t.Fatal("interactive installer")
			}
		}
	}
	if scripts != 1 {
		t.Fatalf("installer executed %d times", scripts)
	}
	f = newFakeCommands()
	f.noInstall = true
	c = NewDevOpsServiceClient(context.Background(), f)
	if err = c.InstallBrew(context.Background()); !executableMissing(err, "brew") {
		t.Fatalf("missing verification: %v", err)
	}
}

func TestSudoPreflightFailsBeforeMutation(t *testing.T) {
	f := newFakeCommands()
	f.sudoError = errors.New("exit status 1")
	c := NewDevOpsServiceClient(context.Background(), f)
	_, err := c.InstallDependencies(context.Background(), []string{"git"})
	if !errors.Is(err, f.sudoError) || len(f.records) != 1 || !reflect.DeepEqual(f.records[0].args, []string{"-n", "true"}) {
		t.Fatalf("%v records=%v", err, f.records)
	}
	if strings.Contains(err.Error(), f.Password()) {
		t.Fatal("password in diagnostic")
	}
}

func TestInstallationFailureIsNotReplayed(t *testing.T) {
	for _, failure := range []string{"install", "plugin"} {
		t.Run(failure, func(t *testing.T) {
			f := newFakeCommands()
			f.tools["brew"] = "/opt/homebrew/bin/brew"
			reset := errors.New("connection lost after exec")
			if failure == "install" {
				f.installError = reset
			} else {
				f.pluginError = reset
			}
			c := NewDevOpsServiceClient(context.Background(), f)
			installed, err := c.InstallDependencies(context.Background(), []string{"vagrant"})
			if !errors.Is(err, reset) {
				t.Fatal(err)
			}
			if failure == "plugin" && !reflect.DeepEqual(installed, []string{"vagrant"}) {
				t.Fatalf("lost ownership: %v", installed)
			}
			count := 0
			for _, r := range f.records {
				if filepath.Base(r.command) == "brew" {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("installer executed %d times", count)
			}
		})
	}
}

func TestProbeResponseAndCache(t *testing.T) {
	for _, response := range []string{"login banner only", "PRL_TOOL_FOUND:", "PRL_TOOL_FOUND:relative/brew", "PRL_TOOL_ABSENT\nPRL_TOOL_FOUND:/opt/homebrew/bin/brew", "PRL_TOOL_FOUND:/opt/homebrew/bin/other"} {
		t.Run(response, func(t *testing.T) {
			f := newFakeCommands()
			f.output = response
			c := NewDevOpsServiceClient(context.Background(), f)
			if _, _, err := c.findPath(context.Background(), "brew"); err == nil {
				t.Fatalf("accepted invalid probe %q", response)
			}
		})
	}
	f := newFakeCommands()
	c := NewDevOpsServiceClient(context.Background(), f)
	_, found, err := c.findPath(context.Background(), "brew")
	if err != nil || found {
		t.Fatal(err)
	}
	f.tools["brew"] = "/opt/homebrew/bin/brew"
	_, found, err = c.findPath(context.Background(), "brew")
	if err != nil || !found {
		t.Fatal("absence was cached")
	}
}

func TestDiscoveryScriptUsesPATHThenFallback(t *testing.T) {
	for _, folder := range []string{"path", "apple", "intel", "home"} {
		t.Run(folder, func(t *testing.T) {
			root := t.TempDir()
			dirs := map[string]string{}
			for _, name := range []string{"path", "apple", "intel", "home"} {
				dirs[name] = filepath.Join(root, name)
				if err := os.Mkdir(dirs[name], 0o700); err != nil {
					t.Fatal(err)
				}
			}
			tool := "bf01-tool"
			expected := filepath.Join(dirs[folder], tool)
			if err := os.WriteFile(expected, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dirs["path"])
			script := strings.NewReplacer("/opt/homebrew/bin", dirs["apple"], "/usr/local/bin", dirs["intel"], "$HOME/bin", dirs["home"]).Replace(discoveryScript)
			out, err := localclient.NewLocalClient().RunCommandContext(context.Background(), "/bin/sh", []string{"-c", script, "discover-tool", tool})
			if err != nil || strings.TrimSpace(out) != "PRL_TOOL_FOUND:"+expected {
				t.Fatalf("%q %v", out, err)
			}
		})
	}
}

func TestOtherDiscoveryConsumersPropagateFailure(t *testing.T) {
	failure := errors.New("transport unavailable")
	checks := map[string]func(*DevOpsServiceClient) error{
		"install PD":   func(c *DevOpsServiceClient) error { return c.InstallParallelsDesktop(context.Background()) },
		"uninstall PD": func(c *DevOpsServiceClient) error { return c.UninstallParallelsDesktop(context.Background()) },
		"install service": func(c *DevOpsServiceClient) error {
			_, err := c.InstallDevOpsService(context.Background(), "", models.ParallelsDesktopDevopsConfigV3{})
			return err
		},
		"uninstall service": func(c *DevOpsServiceClient) error { return c.UninstallDevOpsService(context.Background()) },
		"version":           func(c *DevOpsServiceClient) error { _, err := c.GetDevOpsVersion(context.Background()); return err },
		"license":           func(c *DevOpsServiceClient) error { _, err := c.GetLicense(context.Background()); return err },
	}
	for name, check := range checks {
		t.Run(name, func(t *testing.T) {
			f := newFakeCommands()
			f.probeError = failure
			c := NewDevOpsServiceClient(context.Background(), f)
			if err := check(c); !errors.Is(err, failure) {
				t.Fatal(err)
			}
			if len(f.records) != 1 || f.records[0].command != "/bin/sh" {
				t.Fatalf("mutated after failed probe: %v", f.records)
			}
		})
	}
	f := newFakeCommands()
	f.tools["prlctl"] = "/usr/local/bin/prlctl"
	f.versionError = failure
	if err := NewDevOpsServiceClient(context.Background(), f).InstallParallelsDesktop(context.Background()); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if len(f.records) != 2 {
		t.Fatal("reinstalled after version command failed")
	}
}

func TestLicensePasswordUsesStdin(t *testing.T) {
	f := newFakeCommands()
	f.tools["prlsrvctl"] = "/usr/local/bin/prlsrvctl"
	c := NewDevOpsServiceClient(context.Background(), f)
	secret := "a'\"$`\\; password"
	if err := c.InstallLicense(context.Background(), "license", "account", secret); err != nil {
		t.Fatal(err)
	}
	inputs := 0
	for _, r := range f.records {
		if strings.Contains(strings.Join(r.args, " "), secret) {
			t.Fatal("password in command arguments")
		}
		if r.input != "" {
			inputs++
			if r.input != secret+"\n" {
				t.Fatalf("password altered: %q", r.input)
			}
		}
	}
	if inputs != 1 {
		t.Fatal("password not delivered once")
	}
}

func TestMissingErrorClassification(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", &executableNotFoundError{tool: "prldevops"})
	if !executableMissing(err, "prldevops") || executableMissing(err, "brew") || executableMissing(errors.New("reset"), "prldevops") {
		t.Fatal("incorrect absence classification")
	}
}
