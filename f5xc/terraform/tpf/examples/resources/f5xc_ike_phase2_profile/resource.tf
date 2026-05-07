resource "f5xc_ike_phase2_profile" "example" {
  name      = "example_ike_phase2_profile"
  namespace = "example_namespace"
  ike_key_lifetime = {
    ike_keylifetime_hours = {
      ike_keylifetime_hours = "example_ike_keylifetime_hours"
    }
  }
}
