resource "f5xc_udp_loadbalancer" "example" {
  name      = "example_udp_loadbalancer"
  namespace = "example_namespace"
  loadbalancer_type = {
    udp = true
  }
  advertise_choice = {
    do_not_advertise = true
  }
  hash_policy_choice = {
    hash_policy_choice_round_robin = true
  }
  port_choice = {
    listen_port = 1
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
}
