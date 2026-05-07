resource "f5xc_infraprotect_asn_prefix" "example" {
  namespace = "example_namespace"
  name      = "example_infraprotect_asn_prefix"
  review_type = {
    review_type_pending = true
  }
  irr_status = {
    irr_status_unverified = true
  }
  rpki_status = {
    rpki_status_unverified = true
  }
  asn = {
    name      = "example_infraprotect_asn_prefix"
    namespace = "example_namespace"
  }
  prefix = "example_infraprotect_asn_prefix"
}
