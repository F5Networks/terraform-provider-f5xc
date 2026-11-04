# Example: Origin Pool - Basic

resource "f5xc_origin_pool" "example" {
  name                   = "example-origin-pool"
  namespace              = "system"
  description            = "Origin pool example"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  labels = {
    "env"     = "example"
    "version" = "v1"
  }

  port_choice = {
    port = 8080
  }

  origin_servers = [
    {
      choice = {
        public_name = {
          dns_name = "example.com"
        }
      }
    }
  ]

  tls_choice = {
    no_tls = true
  }

  upstream_conn_pool_reuse_type = {
    map_downstream_to_upstream_conn_pool_type_choice = {
      disable_conn_pool_reuse = true
    }
  }
}
