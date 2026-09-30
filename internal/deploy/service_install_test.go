package deploy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"terraform-provider-parallels-desktop/internal/deploy/models"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"
)

type installerCommands struct {
	*fakeCommands
	version, existing string
	fail              bool
}

func (f *installerCommands) RunCommandContext(ctx context.Context, cmd string, args []string) (string, error) {
	if cmd == "/usr/local/bin/prldevops" {
		return f.version, nil
	}
	if cmd == "sudo" && len(args) > 4 && args[4] == "read-devops-config" {
		return base64.StdEncoding.EncodeToString([]byte(f.existing)), nil
	}
	return f.fakeCommands.RunCommandContext(ctx, cmd, args)
}
func (f *installerCommands) RunCommandInput(ctx context.Context, cmd string, args []string, input io.Reader) (string, error) {
	result, err := f.fakeCommands.RunCommandInput(ctx, cmd, args, input)
	if f.fail {
		return "", errors.New("simulated transaction failure")
	}
	return result, err
}

func TestServiceInstallerUsesCanonicalPrivateConfiguration(t *testing.T) {
	fake := &installerCommands{fakeCommands: newFakeCommands(), version: "prl-devops-service version 1.1.0 (channel: stable)", existing: "environment:\n  STALE: old\nreverse_proxy:\n  enabled: true\n"}
	fake.tools["prldevops"] = "/usr/local/bin/prldevops"
	cfg := models.ParallelsDesktopDevopsConfigV3{RootPassword: types.StringValue("quote'\"$`\npassword"), Port: types.StringValue("9090"), DevOpsVersion: types.StringValue("v1.1.0")}
	version, err := NewDevOpsServiceClient(context.Background(), fake).InstallDevOpsService(context.Background(), "", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.1.0" {
		t.Fatal(version)
	}
	var payload string
	for _, record := range fake.records {
		if strings.Contains(strings.Join(record.args, " "), cfg.RootPassword.ValueString()) {
			t.Fatal("secret in command arguments")
		}
		if record.input != "" {
			payload = record.input
		}
	}
	lines := strings.Split(strings.TrimSpace(payload), "\n")
	if len(lines) != 3 {
		t.Fatal("missing private payload")
	}
	bootstrap, _ := base64.StdEncoding.DecodeString(lines[0])
	var jsonConfig map[string]interface{}
	if err := json.Unmarshal(bootstrap, &jsonConfig); err != nil {
		t.Fatal(err)
	}
	if jsonConfig["root_password"] != cfg.RootPassword.ValueString() {
		t.Fatal("password changed during serialization")
	}
	runtime, _ := base64.StdEncoding.DecodeString(lines[1])
	var doc map[string]interface{}
	if err := yaml.Unmarshal(runtime, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["reverse_proxy"] == nil {
		t.Fatal("lost unrelated configuration")
	}
	env := doc["environment"].(map[string]interface{})
	if env["API_PORT"] != "9090" || env["STALE"] != nil {
		t.Fatal("environment not replaced")
	}
}

func TestVersionMismatchAndBadYAMLDoNotMutateHost(t *testing.T) {
	for _, tc := range []struct{ version, existing, want string }{{"1.0.4", "", "differs"}, {"1.1.0", "[invalid:", "invalid YAML"}} {
		t.Run(tc.want, func(t *testing.T) {
			fake := &installerCommands{fakeCommands: newFakeCommands(), version: tc.version, existing: tc.existing}
			fake.tools["prldevops"] = "/usr/local/bin/prldevops"
			_, err := NewDevOpsServiceClient(context.Background(), fake).InstallDevOpsService(context.Background(), "fallback", models.ParallelsDesktopDevopsConfigV3{DevOpsVersion: types.StringValue("1.1.0")})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatal(err)
			}
			for _, cmd := range fake.records {
				if cmd.input != "" {
					t.Fatal("mutated host after validation failure")
				}
			}
		})
	}
}

func TestVersionOutput(t *testing.T) {
	for _, input := range []string{"1.0.4", "v1.1.0", "prl-devops-service version 1.1.0 (channel: stable)"} {
		if _, err := installedRelease(input); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{"garbage", "1.1.0;echo bad", "main"} {
		if _, err := normalizeRelease(input); err == nil {
			t.Fatal("accepted invalid release")
		}
	}
}

func TestConfigurationTransactionRestoresPreviousState(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(strconv.FormatBool(fail), func(t *testing.T) {
			dir := t.TempDir()
			configDir := filepath.Join(dir, "config")
			if err := os.Mkdir(configDir, 0700); err != nil {
				t.Fatal(err)
			}
			configFile := filepath.Join(configDir, "prldevops_config.yaml")
			plist := filepath.Join(dir, "service.plist")
			loaded := filepath.Join(dir, "loaded")
			previous := []byte("environment:\n  API_PORT: '3080'\n")
			for path, content := range map[string][]byte{configFile: previous, plist: []byte(configFile), loaded: []byte("yes")} {
				if err := os.WriteFile(path, content, 0600); err != nil {
					t.Fatal(err)
				}
			}
			writeTool := func(name, body string) string {
				path := filepath.Join(dir, name)
				if err := os.WriteFile(path, []byte("#!/bin/bash\nset -eu\n"+body), 0700); err != nil {
					t.Fatal(err)
				}
				return path
			}
			writeTool("chown", "exit 0\n")
			installer := writeTool("prldevops", "printf 'legacy installer output' > "+quoteShell(configFile)+"\n"+"printf 'new job' > "+quoteShell(plist)+"\n"+"touch "+quoteShell(loaded)+"\n")
			launch := `case "$1" in
 list) test -f ` + quoteShell(loaded) + `;;
 unload) rm -f ` + quoteShell(loaded) + `;;
 load) if [ -f ` + quoteShell(filepath.Join(dir, "fail")) + ` ]; then rm ` + quoteShell(filepath.Join(dir, "fail")) + `; exit 1; fi; touch ` + quoteShell(loaded) + `;;
 esac
`
			writeTool("launchctl", launch)
			if fail {
				if err := os.WriteFile(filepath.Join(dir, "fail"), []byte("yes"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			script := strings.ReplaceAll(configureServiceScript, "config_dir=/etc/prl-devops-service", "config_dir="+quoteShell(configDir))
			script = strings.ReplaceAll(script, "plist=/Library/LaunchDaemons/com.parallels.prl-devops-service.plist", "plist="+quoteShell(plist))
			if runtime.GOOS != "darwin" {
				script = strings.ReplaceAll(script, "/usr/bin/base64 -D", "/usr/bin/base64 -d")
			}
			payload := base64.StdEncoding.EncodeToString([]byte(`{}`)) + "\n" + base64.StdEncoding.EncodeToString([]byte("environment:\n  API_PORT: '9090'\n")) + "\n" + fmt.Sprintf("%x\n", sha256.Sum256(previous))
			command := exec.Command("/bin/bash", "-c", script, "test-configure", installer)
			command.Env = append(os.Environ(), "PATH="+dir+":/usr/bin:/bin")
			command.Stdin = strings.NewReader(payload)
			output, err := command.CombinedOutput()
			if (err != nil) != fail {
				t.Fatalf("transaction result %v: %s", err, output)
			}
			content, err := os.ReadFile(configFile)
			if err != nil {
				t.Fatal(err)
			}
			if fail && !bytes.Equal(content, previous) {
				t.Fatal("failed reload did not restore previous config")
			}
			if !fail && !strings.Contains(string(content), "9090") {
				t.Fatal("new configuration not installed")
			}
			info, err := os.Stat(configFile)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatal("configuration not private")
			}
			entries, err := os.ReadDir(configDir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("transaction left private artifacts: %v %v", entries, err)
			}
			if _, err := os.Stat(loaded); err != nil {
				t.Fatal("service was not reloaded")
			}
		})
	}
}
