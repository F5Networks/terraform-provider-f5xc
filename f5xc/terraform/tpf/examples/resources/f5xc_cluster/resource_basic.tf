# Example: Cluster - Basic

resource "f5xc_cluster" "example" {
  name      = "example-cluster"
  namespace = "default"

}
