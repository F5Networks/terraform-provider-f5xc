# Example: Workload Flavor - Full

resource "f5xc_workload_flavor" "example" {
  name              = "example-workload-flavor"
  namespace         = "shared"
  description       = "Workload flavor full example"
  memory            = 1024
  vcpus             = 1
  ephemeral_storage = 500

  labels = {
    team = "example-team"
    env  = "example"
  }

  annotations = {
    owner = "platform"
  }
}
