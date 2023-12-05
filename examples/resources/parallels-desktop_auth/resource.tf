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

  # This will create a new API key and user for you
  api_key {
    name   = "example-api-key"
    key    = "example-api-key"
    secret = random_password.api_key.result
  }

  # This will create a new claim for you
  claim {
    name = "EXAMPLE_CLAIM"
  }

  # This will create a new role for you
  role {
    name = "ADMIN"
  }

  # This will create a new user for you
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
