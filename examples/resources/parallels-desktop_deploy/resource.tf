resource "parallels-desktop_deploy" "example" {

  # This will contain the configuration for the Parallels Desktop API
  api_config {
    port                       = "8080"
    prefix                     = "/api"
    log_level                  = "info"
    mode                       = "api"
    devops_version             = "latest"
    root_password              = "VerySecretPassword"
    hmac_secret                = "VerySecretLongStringForHMAC"
    encryption_rsa_key         = "base64 encoded rsa key"
    enable_tls                 = true
    tls_port                   = "8443"
    tls_certificate            = "base64 encoded tls cert"
    tls_private_key            = "base64 encoded tls key"
    disable_catalog_caching    = false
    use_orchestrator_resources = false
    environment_variables = {
      "key" = "value"
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
