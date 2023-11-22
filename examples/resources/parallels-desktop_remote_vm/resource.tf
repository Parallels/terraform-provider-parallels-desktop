resource "parallels-desktop_remote_vm" "example_box" {
  host        = "https://example.com:8080"
  name        = "example"
  owner       = "ec2-user"
  box_name    = "example/fedora-aarch64"
  box_version = "0.0.1"

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

resource "parallels-desktop_remote_vm" "example_vagrant_file" {
  host              = "https://example.com:8080"
  name              = "example"
  owner             = "ec2-user"
  vagrant_file_path = "/path/to/Vagrantfile"

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