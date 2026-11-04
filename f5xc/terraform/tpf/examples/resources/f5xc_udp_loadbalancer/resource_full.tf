# Example: Udp Loadbalancer - Full

resource "f5xc_origin_pool" "example" {
  name                   = "example-udp-lb-full-yxz2jf3d-op"
  namespace              = "default"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  port_choice = {
    port = 53
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


resource "f5xc_udp_loadbalancer" "example" {
  name        = "example-udp-lb-full-yxz2jf3d"
  namespace   = "default"
  description = "udp loadbalancer updated full example"

  labels = {
    env  = "acc"
    team = "network-updated"
  }

  annotations = {
    owner = "terraform-updated"
  }

  domains              = ["udp-acc.example.com", "udp-acc-updated.example.com"]
  dns_volterra_managed = false
  idle_timeout         = 12000
  loadbalancer_type_choice = {
    udp = true
  }
  port_choice = {
    listen_port = 22671
  }
  origin_pools_weights = [
    {
      pool_choice = {
        pool = {
          name      = f5xc_origin_pool.example.name
          namespace = f5xc_origin_pool.example.namespace
        }
      }
      weight   = 3
      priority = 1
    }
  ]
  advertise_choice = {
    advertise_on_public_default_vip = true
  }
  hash_policy_choice = {
    hash_policy_choice_round_robin = true
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
}
