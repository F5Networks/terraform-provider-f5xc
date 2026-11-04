# Example: Tcp Loadbalancer - Basic

resource "f5xc_origin_pool" "example" {
  name                   = "example-tcp-lb-uh3812y3-op"
  namespace              = "terraform-acceptance-test"
  loadbalancer_algorithm = ["ROUND_ROBIN"]
  endpoint_selection     = ["LOCAL_PREFERRED"]

  port_choice = {
    port = 443
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



resource "f5xc_tcp_loadbalancer" "example" {
  name      = "example-tcp-lb-uh3812y3"
  namespace = "terraform-acceptance-test"
  domains   = ["example-tcp-lb-uh3812y3.example.com", "alt-example-tcp-lb-uh3812y3.example.com"]

  origin_pools_weights = [
    {
      pool_choice = {
        pool = {
          name      = f5xc_origin_pool.example.name
          namespace = f5xc_origin_pool.example.namespace
        }
      }
    }
  ]

  port_choice = {
    port_ranges = "24110,24111"
  }

  sni_default_lb_choice = {
    default_lb_with_sni = true
  }

  advertise_choice = {
    advertise_on_public_default_vip = true
  }

  hash_policy_choice = {
    hash_policy_choice_least_active = true
  }

  cluster_retract_choice = {
    retract_cluster = true
  }

  loadbalancer_type_choice = {
    tcp = true
  }

  service_policy_choice = {
    service_policies_from_namespace = true
  }

  description = "acceptance test tcp loadbalancer"
  labels = {
    env  = "example"
    team = "edge"
  }
  annotations = {
    owner = "platform-team"
  }
  idle_timeout = 2000
}
