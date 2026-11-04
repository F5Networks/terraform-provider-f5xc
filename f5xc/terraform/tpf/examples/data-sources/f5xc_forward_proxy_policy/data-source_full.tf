# Example: Forward Proxy Policy - Full

data "f5xc_forward_proxy_policy" "example" {
  name      = f5xc_forward_proxy_policy.example.name
  namespace = f5xc_forward_proxy_policy.example.namespace
}
