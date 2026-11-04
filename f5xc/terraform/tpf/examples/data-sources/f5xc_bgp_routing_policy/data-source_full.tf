# Example: Bgp Routing Policy - Full

data "f5xc_bgp_routing_policy" "example" {
  name      = f5xc_bgp_routing_policy.example.name
  namespace = f5xc_bgp_routing_policy.example.namespace
}
