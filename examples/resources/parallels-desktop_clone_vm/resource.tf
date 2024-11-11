data "parallels-desktop_vm" "example" {
  host = "https:#example.com:8080"

  authenticator {
    username = "john.doe"
    password = "my-password"
  }

  filter {
    field_name       = "name"
    value            = "some-machine-name"
    case_insensitive = true
  }
}

resource "parallels-desktop_clone_vm" "example" {
  # You can only use one of the following options

  # Use the host if you need to connect directly to a host
  host = "http://example.com:8080"
  # Use the orchestrator if you need to connect to a Parallels Orchestrator
  orchestrator = "https://orchestrator.example.com:443"

  name       = "example-vm"
  owner      = "example"
  base_vm_id = data.parallels-desktop_vm.example.machines[count.index].id
  path       = "/some/folder/path"

  # The authenticator block for authenticating to the API, either to the host or orchestrator
  # in this case we are using the API key
  authenticator {
    api_key = "some api key"
  }

  # The configuration for the VM
  config {
    start_headless = true
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

  # this flag will set the desired state for the VM
  # if it is set to true it will keep the VM running otherwise it will stop it
  # by default it is set to true, so all VMs will be running
  keep_running = true

  # This will contain the configuration for the port forwarding reverse proxy
  # in this case we are opening a port to any part in the host, it will not be linked to any
  # specific vm or container. by default it will listen on 0.0.0.0 (all interfaces)
  # and the target host will also be 0.0.0.0 (all interfaces) so it will be open to the world
  # use 
  reverse_proxy_host {
    port = "2022"

    tcp_route {
      target_port = "22"
    }
  }

  # This will contain the configuration for the shared folders
  shared_folder {
    name = "user_download_folder"
    path = "/Users/example/Downloads"
  }

  # This will contain the configuration for the post processor script
  # allowing you to run any command on the VM after it has been deployed
  # you can have multiple lines and they will be executed in order
  post_processor_script {
    # Retry the script 4 times with 10 seconds between each attempt
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
    # Retry the script 4 times with 10 seconds between each attempt
    retry {
      attempts              = 4
      wait_between_attempts = "10s"
    }

    inline = [
      "rm -rf /tmp/*"
    ]
  }
}
