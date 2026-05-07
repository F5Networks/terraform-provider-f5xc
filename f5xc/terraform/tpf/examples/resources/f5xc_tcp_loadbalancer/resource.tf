resource "f5xc_tcp_loadbalancer" "example" {
  name      = "example_tcp_loadbalancer"
  namespace = "example_namespace"
  advertise_choice = {
    advertise_on_public_default_vip = true
  }
  cluster_retract_choice = {
    do_not_retract_cluster = true
  }
  port_choice = {
    listen_port = 1
  }
  sni_default_lb_choice = {
    default_lb_with_sni = true
  }
  hash_policy_choice = {
    hash_policy_choice_least_active = true
  }
  loadbalancer_type = {
    tcp = true
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
}
