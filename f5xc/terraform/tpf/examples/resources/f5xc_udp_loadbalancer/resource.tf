resource "f5xc_udp_loadbalancer" "example" {
  name      = "example_udp_loadbalancer"
  namespace = "example_namespace"
  port_choice = {
    listen_port = 1
  }
  hash_policy_choice = {
    hash_policy_choice_random = true
  }
  loadbalancer_type = {
    udp = true
  }
  service_policy_choice = {
    service_policies_from_namespace = true
  }
  advertise_choice = {
    advertise_on_public_default_vip = true
  }
}
