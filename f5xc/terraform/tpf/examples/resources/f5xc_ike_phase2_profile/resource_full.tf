# Example: Ike Phase2 Profile - Full

resource "f5xc_ike_phase2_profile" "example" {
  name        = "example-ike-phase2-profile"
  namespace   = "default"
  description = "Full IKE phase2 profile example"

  labels = {
    env = "example"
  }

  annotations = {
    owner = "network"
  }

  encryption_algos     = ["AES256_GCM", "AES192_GCM"]
  authentication_algos = ["AUTH_ALG_NONE", "SHA384_HMAC"]

  pfs_mode_choice = {
    dh_group_set = {
      dh_groups = ["DH_GROUP_19", "DH_GROUP_20"]
    }
  }

  ike_key_lifetime_choice = {
    ike_keylifetime_minutes = {
      duration = 180
    }
  }
}
