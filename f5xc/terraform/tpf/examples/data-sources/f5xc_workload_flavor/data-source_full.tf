# Example: Workload Flavor - Full

data "f5xc_workload_flavor" "example" {
  name      = f5xc_workload_flavor.example.name
  namespace = f5xc_workload_flavor.example.namespace
}
