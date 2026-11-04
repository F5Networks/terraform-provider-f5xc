# Example: Bgp Asn Set - Full

resource "f5xc_bgp_asn_set" "example" {
  name        = "example-bgp-asn-set"
  namespace   = "shared"
  description = "BGP ASN set example"
  as_numbers  = [65050, 65051, 65052]

  labels = {
    env  = "acceptance"
    team = "example-team"
  }

  annotations = {
    owner = "netops"
  }
}
