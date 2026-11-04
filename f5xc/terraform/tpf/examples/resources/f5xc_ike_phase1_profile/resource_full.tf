# Example: Ike Phase1 Profile - Full

resource "f5xc_ike_phase1_profile" "example" {
  name        = "example-ike-phase1-profile"
  namespace   = "default"
  description = "Full IKE phase1 profile example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "network"
  }

  encryption_algos     = ["AES256_GCM", "AES192_GCM"]
  authentication_algos = ["AUTH_ALG_NONE", "SHA384_HMAC"]
  prf                  = ["PRFSHA512", "PRFSHA384"]
  dh_group             = ["DH_GROUP_19", "DH_GROUP_20"]

  ike_key_lifetime_choice = {
    ike_keylifetime_minutes = {
      duration = 180
    }
  }

  ike_reauth_timeout_choice = {
    reauth_timeout_days = {
      duration = 10
    }
  }
}
