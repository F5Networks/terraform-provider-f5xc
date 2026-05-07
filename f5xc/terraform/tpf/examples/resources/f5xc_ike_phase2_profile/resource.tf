resource "f5xc_ike_phase2_profile" "example" {
  name      = "example_ike_phase2_profile"
  namespace = "example_namespace"
  ike_key_lifetime = {
    use_default_keylifetime = true
  }
}
