data "parallels-desktop_vm" "example" {
  host = "https://example.com:8080"

  filter {
    field_name       = "name"
    value            = "some-machine-name"
    case_insensitive = true
  }
}

resource "parallels-desktop_clone_vm" "example" {
  host       = "https://example.com:8080"
  name       = "example-vm"
  owner      = "example"
  base_vm_id = data.parallels-desktop_vm.example.machines[count.index].id
  path       = "/some/folder/path"

  authenticator {
    api_key = "some api key"
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