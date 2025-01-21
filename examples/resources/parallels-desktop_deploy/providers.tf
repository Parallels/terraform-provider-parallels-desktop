terraform {
  required_providers {
    parallels-desktop = {
      source = "parallels/parallels-desktop"
    }
  }
}

provider "parallels-desktop" {
  license                = "YOUR_PARALLELS_DESKTOP_LICENSE_KEY"
  disable_tls_validation = true
}
