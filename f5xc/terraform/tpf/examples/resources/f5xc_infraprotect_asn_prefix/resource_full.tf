# Example: Infraprotect ASN Prefix - Full

# The referenced ASN must already be approved in the F5 XC tenant.
resource "f5xc_infraprotect_asn_prefix" "example" {
  name        = "example-infraprotect-asn-prefix"
  namespace   = "system"
  description = "DDoS transit prefix associated with an approved ASN"

  labels = {
    env  = "example"
    team = "network-security"
  }

  annotations = {
    owner = "platform-team"
  }

  asn = {
    name      = "approved-asn"
    namespace = "system"
  }

  prefix = "9.9.9.0/24"
}