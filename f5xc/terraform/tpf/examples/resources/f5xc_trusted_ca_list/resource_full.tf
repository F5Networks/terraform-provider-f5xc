# Example: Trusted CA List - Full

resource "f5xc_trusted_ca_list" "example" {
  name           = "example-ug-zemh7i0q"
  namespace      = "default"
  description    = "Full test trusted CA list"
  trusted_ca_url = "https://example.com/ca.pem"

  labels = {
    env        = "example"
    managed-by = "terraform"
  }

  annotations = {
    owner = "acc-test"
  }
}