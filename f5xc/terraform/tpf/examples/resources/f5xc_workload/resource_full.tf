# Example: Workload - Full

resource "f5xc_workload" "example" {
  name        = "example-workload"
  namespace   = "default"
  description = "Workload full example"
  labels = {
    env = "example"
  }
  annotations = {
    owner = "platform"
  }

  workload_choice = {
    simple_service = {
      container = {
        name = "web"

        image = {
          name = "nginx:1.25"
          registry_choice = {
            public = true
          }
          pull_policy = ["IMAGE_PULL_POLICY_NEVER"]
        }
        flavor_choice = {
          flavor = ["CONTAINER_FLAVOR_TYPE_LARGE"]
        }
      }
      configuration = {
        parameters = [{
          choice = {
            env_var = {
              name  = "APP_MODE"
              value = "production"
            }
          }
        }]
      }
      advertise_choice = {
        do_not_advertise = true
      }
      persistence_choice = {
        disabled = true
      }
      scale_to_zero = true
    }
  }
}
