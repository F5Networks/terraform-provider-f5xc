resource "f5xc_workload" "example" {
  name      = "example_workload"
  namespace = "example_namespace"
  workload_choice = {
    service = {
      service = "example_service"
    }
  }
}
