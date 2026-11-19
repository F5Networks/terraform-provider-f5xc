# Example: Cluster - Full

resource "f5xc_cluster" "example" {
  name      = "example-cluster"
  namespace = "default"
  description = "Cluster example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "platform"
  }

  endpoint_selection = ["DISTRIBUTED"]
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  connection_timeout = 2000

  max_requests_per_connection_choice = {
    max_requests_per_connection = 200
  }

  upstream_conn_pool_reuse_type = {
    map_downstream_to_upstream_conn_pool_type_choice = {
      enable_conn_pool_reuse = true
    }
  }

  endpoint_subsets = [
    {
      keys = ["site", "stage"]
    }
  ]

}
