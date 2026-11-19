# Example: Registration Approval

# The matching registration must already exist in the PENDING state in the
# system namespace. Approval changes its state to APPROVED.
resource "f5xc_registration_approval" "example" {
  namespace    = "system"
  cluster_name = "example-cluster"
  hostname     = "example-host"
  cluster_size = 1

  retry       = 5
  wait_time   = 30
  tunnel_type = ["SITE_TO_SITE_TUNNEL_IPSEC_OR_SSL"]
}
