# Example: Dc Cluster Group - Full

data "f5xc_dc_cluster_group" "example" {
  name      = f5xc_dc_cluster_group.example.name
  namespace = f5xc_dc_cluster_group.example.namespace
}
