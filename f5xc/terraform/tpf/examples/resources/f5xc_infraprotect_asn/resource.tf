resource "f5xc_infraprotect_asn" "example" {
  name      = "example_infraprotect_asn"
  namespace = "example_namespace"
  asn       = 1
  bgp_session = {
    bgp_session_disabled = true
  }
  review_type = {
    review_type_pending = true
  }
}
