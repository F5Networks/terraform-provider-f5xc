# Example: Dc Cluster Group - Full

resource "f5xc_dc_cluster_group" "example" {
  name        = "example-dc-cluster-group-idu7oe5n"
  namespace   = "system"
  description = "DC cluster group example"

  labels = {
    env  = "example"
    tier = "regional"
  }

  annotations = {
    owner = "platform"
  }

  type = {
    dc_cluster_group_mesh_choice = {
      data_plane_mesh = true
    }
  }
}
