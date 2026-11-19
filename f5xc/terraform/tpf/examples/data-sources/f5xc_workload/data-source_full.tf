# Example: Workload - Full

data "f5xc_workload" "example" {
  name      = f5xc_workload.example.name
  namespace = f5xc_workload.example.namespace
}
