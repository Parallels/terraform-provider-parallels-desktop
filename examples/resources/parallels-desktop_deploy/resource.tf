resource "parallels-desktop_deploy" "example" {

  # This will contain the configuration for the Parallels Desktop API
  api_config {
    port   = "8080"
    prefix = "/api"
    # This will set the log level for the API
    log_level = "info"
    # This will enable logging for the API
    enable_logging = true
    # This will set the mode for the API, you can use either api or orchestrator. by default it will be api
    mode = "api"
    # you can force any version of the devops api, if you leave it empty it will use the latest version
    # but it will not automatically update to the latest version, that would need a manual step
    devops_version = "latest"
    # This will set the password for the default root user
    root_password = "VerySecretPassword"
    # This will enable the api to use the hmac secret for the authentication
    hmac_secret = "VerySecretLongStringForHMAC"
    # This will enable the api to use the rsa key for encryption of the database file
    # we strongly advise you to use this feature for security reasons
    encryption_rsa_key = "base64 encoded rsa key"
    # This will enable the api to use the tls certificate
    enable_tls = true
    # This will enable the tls port
    tls_port = "8443"
    # This will enable the tls certificate
    tls_certificate = "base64 encoded tls cert"
    # This will enable the tls private key
    tls_private_key = "base64 encoded tls key"
    # This will enable the catalog caching, this will cache the catalog in the host
    disable_catalog_caching = false
    # This will enable the orchestrator resources, this will enable the host to use the orchestrator
    use_orchestrator_resources = false
    # This will enable the port forwarding reverse proxy in the host, you will need to set the
    # port_forwarding block to configure the ports in the deploy or any other provider
    enable_port_forwarding = false
    # This will allow more fine tune of the api configuration, you can pass any compatible environment
    # variable 
    environment_variables = {
      "key" = "value"
    }
  }

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

  # This will contain the configuration for the Parallels Desktop Orchestrator
  # and how to register this instance with it
  orchestrator_registration {
    tags = [
      "any identifying tags for this instance"
    ]
    description = "some description for this instance"

    # this block will contain the configuration for the orchestrator
    orchestrator {
      schema = "http"
      host   = "example.com"
      port   = "80"
      authentication {
        username = "user@example.com"
        password = "VerySecretPassword"
      }
    }
  }

  # This will contain the ssg configuration required to deploy Parallels Desktop to the remote
  # machine, if you are deploying this locally then you can omit this block and use the flag instead
  # but we will not be able to install all required dependencies so we still advise you to use this
  # and set a ssh server up on the local machine
  ssh_connection {
    type        = "ssh"
    user        = "ec2-user"
    private_key = "the private ssh key"
    host        = "example.com"
    host_port   = "22"
  }
}
