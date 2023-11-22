resource "random_password" "api_key" {
  length           = 32
  special          = false
  override_special = "_%@"
}

resource "random_password" "admin_user" {
  length           = 8
  special          = false
  override_special = "_%@"
}

resource "parallels-desktop_auth" "example" {
  host = "http://example.com:8080"

  api_key {
    name   = "example-api-key"
    key    = "example-api-key"
    secret = random_password.api_key.result
  }

  claim {
    name = "EXAMPLE_CLAIM"
  }

  role {
    name = "ADMIN"
  }

  user {
    name     = "Admin User"
    username = "admin"
    email    = "admin@example.com"
    password = random_password.admin_user.result
    roles = [
      {
        name = "ADMIN"
      }
    ]
    claims = [
      {
        name = "EXAMPLE_CLAIM"
      }
    ]
  }
}
