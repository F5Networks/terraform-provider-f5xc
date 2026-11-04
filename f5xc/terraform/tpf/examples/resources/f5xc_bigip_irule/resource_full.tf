# Example: Bigip Irule - Full

resource "f5xc_bigip_irule" "example" {
  name        = "example-bigip-irule"
  namespace   = "default"
  description = "BIG-IP iRule example"
  source      = "AI"
  irule_name  = "full-irule"
  code        = "when CLIENT_ACCEPTED { log local0. client-accepted }"

  labels = {
    env  = "example"
    tier = "shared"
  }

  annotations = {
    owner = "platform"
    team  = "edge"
  }
}
