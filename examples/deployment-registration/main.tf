# Requires a provider build containing BF-02 and BF-03.
terraform {
  required_providers {
    parallels-desktop = {
      source = "Parallels/parallels-desktop"
    }
  }
}

variable "license" {
  type      = string
  sensitive = true
}
variable "mac_host" {
  type = string
}
variable "ssh_user" {
  type = string
}
variable "ssh_private_key" {
  type      = string
  sensitive = true
}
variable "host_root_password" {
  type      = string
  sensitive = true
}
variable "register_with_orchestrator" {
  type    = bool
  default = false
}
variable "orchestrator_url" {
  type    = string
  default = null
}
variable "orchestrator_api_key" {
  type      = string
  sensitive = true
  default   = null
}

provider "parallels-desktop" {
  license = var.license
}

resource "parallels-desktop_deploy" "mac" {
  ssh_connection {
    host        = var.mac_host
    user        = var.ssh_user
    private_key = var.ssh_private_key
  }

  api_config {
    devops_version  = "1.1.0"
    port            = "8080"
    prefix          = "/api"
    enabled_modules = ["api", "host"]
    root_password   = var.host_root_password
  }

  # This boolean belongs to this example/module, not to the native resource.
  # false omits the block; true creates it with separate Orchestrator credentials.
  dynamic "orchestrator_registration" {
    for_each = var.register_with_orchestrator ? [1] : []
    content {
      description = "Terraform-managed Mac"
      orchestrator {
        host = var.orchestrator_url
        authentication {
          api_key = var.orchestrator_api_key
        }
      }
    }
  }
}

output "registration" {
  value = {
    registered = parallels-desktop_deploy.mac.is_registered_in_orchestrator
    host       = parallels-desktop_deploy.mac.orchestrator_host
    host_id    = parallels-desktop_deploy.mac.orchestrator_host_id
  }
}
