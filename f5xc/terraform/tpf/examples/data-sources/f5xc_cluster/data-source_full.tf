# Example: Cluster - Full

data "f5xc_cluster" "example" {
  name      = f5xc_cluster.example.name
  namespace = f5xc_cluster.example.namespace
}
