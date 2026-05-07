resource "f5xc_fast_acl" "example" {
  name             = "example_fast_acl"
  namespace        = "example_namespace"
  network_type     = {}
  destination_type = {}
}
