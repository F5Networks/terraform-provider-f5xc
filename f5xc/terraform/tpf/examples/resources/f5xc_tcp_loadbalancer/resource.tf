resource "f5xc_tcp_loadbalancer" "example" {
  name      = "example_tcp_loadbalancer"
  namespace = "example_namespace"
  loadbalancer_type = {
    tls_tcp_auto_cert = {
      tls_tcp_auto_cert = "example_tls_tcp_auto_cert"
    }
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
  hash_policy_choice = {
    hash_policy_choice_random = true
  }
  sni_default_lb_choice = {
    default_lb_with_sni = true
  }
  advertise_choice = {
    do_not_advertise = true
  }
  port_choice = {
    listen_port = 1
  }
  cluster_retract_choice = {
    do_not_retract_cluster = true
  }
}
