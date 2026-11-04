# Example: Infraprotect ASN - Full

resource "f5xc_infraprotect_asn" "example" {
  name        = "example-infraprotect-asn"
  namespace   = "system"
  description = "ASN for Infraprotect DDoS transit protection"
  asn         = 4199999999

  labels = {
    env  = "example"
    team = "network-security"
  }

  annotations = {
    owner = "platform-team"
  }

  bgp_session_choice = {
    bgp_session_enabled = true
  }
}