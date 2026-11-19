# Example: Advertise Policy - Full

data "f5xc_advertise_policy" "example" {
  name      = f5xc_advertise_policy.example.name
  namespace = f5xc_advertise_policy.example.namespace
}
