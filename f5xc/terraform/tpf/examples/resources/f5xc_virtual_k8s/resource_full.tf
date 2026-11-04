# Example: Virtual K8s - Full

resource "f5xc_namespace" "test_namespace" {
  name = "example-namespace"
}

resource "f5xc_virtual_k8s" "example" {
  name        = "example-virtual-k8s"
  namespace   = f5xc_namespace.test_namespace.name
  description = "Virtual k8s full example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "example-team"
  }

  service_isolation_choice = {
    disabled = true
  }
}
