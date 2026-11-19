# Example: Network Firewall - Forward Proxy And Fast Acl Dependencies

resource "f5xc_forward_proxy_policy" "dep" {
  name      = "example-network-firewall-65yn2k-922000-fpp"
  namespace = "system"

  proxy_choice = {
    any_proxy = true
  }

  rule_choice = {
    allow_all = true
  }
}

resource "f5xc_fast_acl" "dep" {
  name      = "example-network-firewall-65yn2k-922000-fast-acl"
  namespace = "system"

  site_choice = {
    site_acl = {
      network_choice = {
        outside_network = true
      }
      vip_choice = {
        all_services = true
      }
    }
  }
}

resource "f5xc_network_firewall" "example" {
  name      = "example-network-firewall"
  namespace = "system"

  description = "Network firewall example depending on forward proxy and fast ACL"

  annotations = {
    forward_proxy_dependency = f5xc_forward_proxy_policy.dep.name
    fast_acl_dependency      = f5xc_fast_acl.dep.name
  }

  depends_on = [
    f5xc_forward_proxy_policy.dep,
    f5xc_fast_acl.dep,
  ]
}
