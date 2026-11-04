# Example: Tcp Loadbalancer Service Policy Attachments - Basic

resource "f5xc_origin_pool" "example" {
  name                   = "tf-acc-sp-op-e9ewpvro"
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


resource "f5xc_service_policy" "sp1" {
  name          = "tf-acc-sp-sp1-e9ewpvro"
  namespace     = "terraform-acceptance-test"
  rule_choice   = { allow_all_requests = true }
  server_choice = { any_server = true }
}

resource "f5xc_service_policy" "sp2" {
  name          = "tf-acc-sp-sp2-e9ewpvro"
  namespace     = "terraform-acceptance-test"
  rule_choice   = { allow_all_requests = true }
  server_choice = { any_server = true }
}

resource "f5xc_tcp_loadbalancer" "example" {
  name      = "example-tcp-lb-att-e9ewpvro"
  namespace = "terraform-acceptance-test"

  port_choice              = { listen_port = 21386 }
  advertise_choice         = { do_not_advertise = true }
  hash_policy_choice       = { hash_policy_choice_round_robin = true }
  cluster_retract_choice   = { do_not_retract_cluster = true }
  loadbalancer_type_choice = { tcp = true }
  sni_default_lb_choice    = { no_sni = true }

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
}

resource "f5xc_tcp_loadbalancer_service_policy_attachments" "example" {
  namespace             = "terraform-acceptance-test"
  tcp_loadbalancer_name = f5xc_tcp_loadbalancer.example.name
  service_policies      = [
    f5xc_service_policy.sp1.name,
    f5xc_service_policy.sp2.name,
  ]
}
