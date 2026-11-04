# Example: Container Registry - Full

resource "f5xc_container_registry" "example" {
  name        = "example-container-registry"
  namespace   = "shared"
  description = "Container registry example"
  registry    = "registry.example.com"
  user_name   = "registry-user"
  email       = "registry@example.com"

  labels = {
    env  = "example"
    tier = "shared"
  }

  annotations = {
    owner = "platform"
    team  = "security"
  }

  password = {
    secret_info_choice = {
      clear_secret_info = {
        url = "string:///Y29udGFpbmVyLXJlZ2lzdHJ5LXBhc3M="
      }
    }
  }
}
