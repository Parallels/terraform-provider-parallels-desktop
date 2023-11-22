resource "parallels-desktop_deploy" "example" {
  api_config {
    port                       = "8080"
    prefix                     = "/api"
    log_level                  = "info"
    mode                       = "api"
    install_version            = "latest"
    root_password              = "VerySecretPassword"
    hmac_secret                = "VerySecretLongStringForHMAC"
    encryption_rsa_key         = "base64 encoded rsa key"
    enable_tls                 = true
    tls_port                   = "8443"
    tls_certificate            = "base64 encoded tls cert"
    tls_private_key            = "base64 encoded tls key"
    disable_catalog_caching    = false
    use_orchestrator_resources = false
  }

  ssh_connection = {
    type        = "ssh"
    user        = "ec2-user"
    private_key = "the private ssh key"
    host        = "example.com"
    host_port   = "22"
  }
}