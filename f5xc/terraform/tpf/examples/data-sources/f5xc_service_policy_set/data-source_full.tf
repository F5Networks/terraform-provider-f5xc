# Example: Service Policy Set - Full

data "f5xc_service_policy_set" "example" {
  name      = "default"
  namespace = "default"
}
