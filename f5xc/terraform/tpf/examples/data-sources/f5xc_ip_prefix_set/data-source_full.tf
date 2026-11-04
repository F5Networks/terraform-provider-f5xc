# Example: Ip Prefix Set - Full

data "f5xc_ip_prefix_set" "example" {
  name      = f5xc_ip_prefix_set.example.name
  namespace = f5xc_ip_prefix_set.example.namespace
}
