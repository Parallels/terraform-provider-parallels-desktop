---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "parallels-desktop_remote_vm Resource - terraform-provider-parallels-desktop"
subcategory: ""
description: |-
  Parallels Virtual Machine State Resource
---

# parallels-desktop_remote_vm (Resource)

Parallels Virtual Machine State Resource

## Example Usage

```terraform
resource "parallels-desktop_remote_vm" "example_box" {
  host            = "https://example.com:8080"
  name            = "example-vm"
  owner           = "example"
  catalog_id      = "example-catalog-id"
  version         = "v1"
  host_connection = "host=user:VerySecretPassword@example.com"
  path            = "/Users/example/Parallels"

  # This will tell how should we authenticate with the host API
  # you can either use it or leave it empty, if left empty then
  # we will use the default root user and password
  authenticator {
    api_key = "host api key"
  }

  # This will contain some common configuration for the VM
  # like if we should start it headless or not
  config {
    start_headless     = false
    enable_rosetta     = false
    pause_idle         = false
    auto_start_on_host = false
  }

  # This will contain a configuration for the specs of the VM
  specs {
    cpu_count   = "2"
    memory_size = "2048"
  }

  # this will allow you to fine grain the configuration of the VM
  # you can pass any command that is compatible with the prlctl command
  # directly to the VM
  # Attention: the prlctl will not keep the state, meaning it will always
  # execute the action and if you remove it it will not bring the machine
  # to the previous state before setting that configuration
  prlctl {
    operation = "set"
    flags = [
      "some-flag"
    ]

    options = [
      {
        flag  = "description"
        value = "some description"
      }
    ]
  }

  force_changes = true

  # This will contain the configuration for the shared folders
  shared_folder {
    name = "user_download_folder"
    path = "/Users/example/Downloads"
  }

  # This will contain the configuration for the post processor script
  # allowing you to run any command on the VM after it has been deployed
  # you can have multiple lines and they will be executed in order
  post_processor_script {
    // Retry the script 4 times with 10 seconds between each attempt
    retry {
      attempts              = 4
      wait_between_attempts = "10s"
    }

    inline = [
      "ls -la"
    ]
  }

  # This is a special block that will allow you to undo any changes your scripts have done
  # if you are destroying a VM, like unregistering from a service where the VM was registered
  on_destroy_script {
    // Retry the script 4 times with 10 seconds between each attempt
    retry {
      attempts              = 4
      wait_between_attempts = "10s"
    }

    inline = [
      "rm -rf /tmp/*"
    ]
  }
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `catalog_connection` (String) Parallels DevOps Catalog Connection
- `catalog_id` (String) Catalog Id to pull
- `name` (String) Virtual Machine name to create, this needs to be unique in the host
- `path` (String) Path

### Optional

- `architecture` (String) Virtual Machine architecture
- `authenticator` (Block, Optional) Authenticator block, this is used to authenticate with the Parallels Desktop API, if empty it will try to use the root password (see [below for nested schema](#nestedblock--authenticator))
- `config` (Block, Optional) Virtual Machine config block, this is used set some of the most common settings for a VM (see [below for nested schema](#nestedblock--config))
- `force_changes` (Boolean) Force changes, this will force the VM to be stopped and started again
- `host` (String) Parallels Desktop DevOps Host
- `on_destroy_script` (Block List) Run any script after the virtual machine is created (see [below for nested schema](#nestedblock--on_destroy_script))
- `orchestrator` (String) Parallels Desktop DevOps Orchestrator
- `owner` (String) Virtual Machine owner
- `post_processor_script` (Block List) Run any script after the virtual machine is created (see [below for nested schema](#nestedblock--post_processor_script))
- `prlctl` (Block List) Virtual Machine config block, this is used set some of the most common settings for a VM (see [below for nested schema](#nestedblock--prlctl))
- `run_after_create` (Boolean) Run after create, this will make the VM to run after creation
- `shared_folder` (Block List) Shared Folders Block, this is used to share folders with the virtual machine (see [below for nested schema](#nestedblock--shared_folder))
- `specs` (Block, Optional) Virtual Machine Specs block, this is used to set the specs of the virtual machine (see [below for nested schema](#nestedblock--specs))
- `timeouts` (Attributes) (see [below for nested schema](#nestedatt--timeouts))
- `version` (String) Catalog version to pull, if empty will pull the 'latest' version

### Read-Only

- `id` (String) Virtual Machine Id
- `os_type` (String) Virtual Machine OS type

<a id="nestedblock--authenticator"></a>
### Nested Schema for `authenticator`

Optional:

- `api_key` (String, Sensitive) Parallels desktop API API Key
- `password` (String, Sensitive) Parallels desktop API Password
- `username` (String) Parallels desktop API Username


<a id="nestedblock--config"></a>
### Nested Schema for `config`

Optional:

- `auto_start_on_host` (Boolean) Start the VM when the host starts, this will stop the VM if it is running
- `enable_rosetta` (Boolean) Enable Rosetta on Apple Silicon, this will stop the VM if it is running
- `pause_idle` (Boolean) Pause the VM when the host is idle, this will stop the VM if it is running
- `start_headless` (Boolean) Set the VM to start headless, this will stop the VM if it is running


<a id="nestedblock--on_destroy_script"></a>
### Nested Schema for `on_destroy_script`

Optional:

- `inline` (List of String) Inline script
- `retry` (Block, Optional) Retry settings (see [below for nested schema](#nestedblock--on_destroy_script--retry))

Read-Only:

- `result` (Attributes List, Sensitive) Result of the script (see [below for nested schema](#nestedatt--on_destroy_script--result))

<a id="nestedblock--on_destroy_script--retry"></a>
### Nested Schema for `on_destroy_script.retry`

Optional:

- `attempts` (Number) Number of attempts
- `wait_between_attempts` (String) Wait between attempts, you can use the suffixes 's' for seconds, 'm' for minutes


<a id="nestedatt--on_destroy_script--result"></a>
### Nested Schema for `on_destroy_script.result`

Optional:

- `exit_code` (String) Exit code
- `script` (String) Script
- `stderr` (String) Stderr
- `stdout` (String) Stdout



<a id="nestedblock--post_processor_script"></a>
### Nested Schema for `post_processor_script`

Optional:

- `inline` (List of String) Inline script
- `retry` (Block, Optional) Retry settings (see [below for nested schema](#nestedblock--post_processor_script--retry))

Read-Only:

- `result` (Attributes List, Sensitive) Result of the script (see [below for nested schema](#nestedatt--post_processor_script--result))

<a id="nestedblock--post_processor_script--retry"></a>
### Nested Schema for `post_processor_script.retry`

Optional:

- `attempts` (Number) Number of attempts
- `wait_between_attempts` (String) Wait between attempts, you can use the suffixes 's' for seconds, 'm' for minutes


<a id="nestedatt--post_processor_script--result"></a>
### Nested Schema for `post_processor_script.result`

Optional:

- `exit_code` (String) Exit code
- `script` (String) Script
- `stderr` (String) Stderr
- `stdout` (String) Stdout



<a id="nestedblock--prlctl"></a>
### Nested Schema for `prlctl`

Optional:

- `flags` (List of String) Set the VM flags, this will stop the VM if it is running
- `operation` (String) Set the VM to start headless, this will stop the VM if it is running
- `options` (Attributes List) Set the VM options, this will stop the VM if it is running (see [below for nested schema](#nestedatt--prlctl--options))

<a id="nestedatt--prlctl--options"></a>
### Nested Schema for `prlctl.options`

Optional:

- `flag` (String) Set the VM option flag, this will stop the VM if it is running
- `value` (String) Set the VM option value, this will stop the VM if it is running



<a id="nestedblock--shared_folder"></a>
### Nested Schema for `shared_folder`

Optional:

- `description` (String) Description
- `disabled` (Boolean) Disabled
- `name` (String) Shared folder name
- `path` (String) Path to share
- `readonly` (Boolean) Read only


<a id="nestedblock--specs"></a>
### Nested Schema for `specs`

Optional:

- `cpu_count` (String) The number of CPUs of the virtual machine.
- `disk_size` (String) The size of the disk of the virtual machine in megabytes.
- `force` (Boolean) Force the specs to be set, this will stop the VM if it is running
- `memory_size` (String) The amount of memory of the virtual machine in megabytes.


<a id="nestedatt--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
