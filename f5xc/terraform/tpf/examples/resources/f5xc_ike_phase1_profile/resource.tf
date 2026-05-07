resource "f5xc_ike_phase1_profile" "example" {
  name      = "example_ike_phase1_profile"
  namespace = "example_namespace"
  ike_key_lifetime = {
    ike_keylifetime_hours = {
      ike_keylifetime_hours = "example_ike_keylifetime_hours"
    }
  }
  ike_reauth_timeout = {
    reauth_timeout_hours = {
      reauth_timeout_hours = "example_reauth_timeout_hours"
    }
  }
}
