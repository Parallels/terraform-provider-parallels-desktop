data "parallels-desktop_vm" "example" {
  # You can only use one of the following options

  # Use the host if you need to connect directly to a host
  host = "http:#example.com:8080"
  # Use the orchestrator if you need to connect to a Parallels Orchestrator
  orchestrator = "https:#orchestrator.example.com:443"

  # The authenticator block for authenticating to the API, either to the host or orchestrator
  authenticator {
    username = "john.doe"
    password = "my-password"
  }

  # The filter block to filter the VMs
  filter {
    field_name = "name"
    value      = "exampe-vm"
  }
}
