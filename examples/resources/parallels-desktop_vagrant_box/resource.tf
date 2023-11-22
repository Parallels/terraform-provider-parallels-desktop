resource "parallels-desktop_remote_vm" "example" {
  host            = "https://example.com:8080"
  name            = "example"
  owner           = "ec2-user"
  catalog_id      = "example-machine"
  host_connection = "host=user:password@catalog.example.com"
  path            = "/Users/example/Parallels"

  authenticator {
    api_key = "host api key"
  }

  specs {
    cpu_count   = "2"
    memory_size = "2048"
  }

  force_changes = true

  shared_folder {
    name = "user_download_folder"
    path = "/Users/example/Downloads"
  }

  post_processor_script {
    inline = [
      "ls -la"
    ]
  }
}